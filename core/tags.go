package core

import ()

//Tagging is integral to this product.
//Tags are stored in the store like anything else.
//by convention tags are a filepath-like string.
//this is used to build hierachy for e.g. exporting a dir structure.
// e.g. `tags/awesome`, `albums/some-album`, `people/chris`, `events/xyz`
// or whatever.
//but the API adds/removes tags, and those tags are also in the store.
//so we some functions for adding/removing from items.
//
// to add/remove tags, we need to get the item, change the tags array
// then save AND re-index.
// for any added tags we should check that they exist first and add them
// to the store/index before updating the items.

// @TODO add new tags to store.
func SetItemTags(s *Service, item *Item, tags ...string) error {
	item.Tags = tags
	return item_update(s, item)
}

//find all items with tag and build a map[hash]bool with all false
//then iterate hashes and mark them all true.
//then iterate over the map and delete the tag from items with false,
//add tag to items with true.
func SetTagItems(s *Service, tag string, hashes ...string) chan error {
	return make(chan error)
}

//add a single tag to an item.
func AddTagToItem(s *Service, item *Item, tag string) error {
	for _, t := range item.Tags {
		if t == tag {
			return nil
		}
	}
	//doesn't exist.
	item.Tags = append(item.Tags, tag)
	return item_update(s, item)
}

//remove a single tag from an item.
func RemoveTagFromItem(s *Service, item *Item, tag string) error {
	if len(item.Tags) == 0 {
		return nil
	}
	newTags := make([]string, 0, len(item.Tags))
	for _, t := range item.Tags {
		if t != tag {
			newTags = append(newTags, t)
		}
	}
	if len(newTags) == len(item.Tags) {
		//nothing changed.
		return nil
	}
	item.Tags = newTags
	return item_update(s, item)
}
