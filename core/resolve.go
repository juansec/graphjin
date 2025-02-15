package core

import (
	"fmt"
	"strings"

	"github.com/dosco/graphjin/core/internal/sdata"
)

type resFn func(v ResolverProps) (Resolver, error)

type resItem struct {
	IDField []byte
	Path    [][]byte
	Fn      Resolver
}

func (gj *GraphJin) initResolvers() error {
	gj.rmap = make(map[string]resItem)

	err := gj.conf.SetResolver("remote_api", func(v ResolverProps) (Resolver, error) {
		return newRemoteAPI(v)
	})

	if err != nil {
		return err
	}

	for _, r := range gj.conf.Resolvers {
		if err := gj.initRemote(r); err != nil {
			return fmt.Errorf("resolvers: %w", err)
		}
	}

	return nil
}

func (gj *GraphJin) initRemote(rc ResolverConfig) error {
	// Defines the table column to be used as an id in the
	// remote reques
	var col sdata.DBColumn

	ti, err := gj.schema.GetTableInfo(rc.Table, "")
	if err != nil {
		return err
	}

	// If no table column specified in the config then
	// use the primary key of the table as the id
	if rc.Column != "" {
		idcol, err := ti.GetColumn(rc.Column)
		if err != nil {
			return err
		}
		col = idcol
	} else {
		col = ti.PrimaryCol
	}

	idk := fmt.Sprintf("__%s_%s", rc.Name, col.Name)

	// Register a relationship between the remote data
	// and the database table
	val := sdata.DBRel{Type: sdata.RelRemote}
	val.Left.Col = col
	val.Right.VTable = idk

	if err := gj.schema.SetRel(rc.Name, rc.Table, val, false); err != nil {
		return err
	}

	// The function thats called to resolve this remote
	// data request
	var fn Resolver

	if v, ok := gj.conf.rtmap[rc.Type]; ok {
		fn, err = v(rc.Props)
	} else {
		err = fmt.Errorf("unknown resolver type: %s", rc.Type)
	}

	if err != nil {
		return err
	}

	path := [][]byte{}
	for _, p := range strings.Split(rc.StripPath, ".") {
		path = append(path, []byte(p))
	}

	rf := resItem{
		IDField: []byte(idk),
		Path:    path,
		Fn:      fn,
	}

	// Index resolver obj by parent and child names
	gj.rmap[(rc.Name + rc.Table)] = rf

	// Index resolver obj by IDField
	gj.rmap[string(rf.IDField)] = rf

	return nil
}
