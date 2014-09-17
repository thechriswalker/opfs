// this is how the app state looks
console.log({
  //the url for routing purposes.
  url: {
    full: "http://some.domain/some/path?query#fragment",
    path: "/some/path",
    query: {"query": null},
    fragment: "fragment"
  },
  //the current page title
  title: "current page title",
  //if present, and >= 0, the index of the current slideshow item in the items list.
  slideshow: -1,
  //the current items we have (in order!)
  items: [
    "sha1-xyz",
    "sha1-xyz",
    "sha1-xyz",
    "sha1-xyz",
  ],
  paging: {
    //total number of hits for this search
    total: 1024,
    last: "/api/search?query=...", //last search query ran
    next: "/api/search?query=...", //next page of the current search (or null)
    error: "error fecthing page" //error message or null, designed to represent an error fetching a page.
  },
  //key value version of "items"
  objects: {
    "sha1-xyz": {
      Hash: "sha1-xyz",
      // ...
    },
    // ...
  }
});