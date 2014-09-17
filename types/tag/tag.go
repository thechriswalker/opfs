package tag

import (
	"bufio"
	"encoding/json"
	"io"

	//our packages
	"code.7r.pm/chris/opfs/core"
)

const TAG_SIZE_MAX = 512 //512 bytes for a tag, pretty good.

type Tag struct {
	Slug string
}

func (p *Tag) EnsureMeta(i *core.Item) error {
	if p.Slug == "" {
		//only do it if we have to
		return json.Unmarshal(i.Meta, p)
	}
	return nil
}

//We should never have to inspect this...
func (t *Tag) Inspect(r io.ReadSeeker) (item *core.Item, err error) {
	item = &core.Item{
		Type: core.ItemTypeTag,
	}
	//r should just contain the slug
	bufrd := bufio.NewReaderSize(r, TAG_SIZE_MAX)
	name, _, err := bufrd.ReadLine()
	if err != nil {
		return nil, err
	}
	t.Slug = string(name)

	b, err := json.Marshal(t)
	if err != nil {
		return nil, err
	}

	//now add meta to item.
	item.Meta = json.RawMessage(b)

	return
}
