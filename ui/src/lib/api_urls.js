var path = require("path"),
    lodash = require("lodash");

var mediaTypes = ["Video","Photo"],
    tagTypes = ["Tag"];

var
  url = function(){
    var args = Array.prototype.slice.call(arguments);
    args.unshift("/api");
    return path.join.apply(path, args);
  },
  enc = encodeURIComponent,
  stringifyQueryString = function(q){
    return lodash.transform(q, function(a, v, k){
      a.push(enc(k)+"="+enc(v));
    }, []).join("&");
  },
  encSearch = function(obj){
    return stringifyQueryString({"query": JSON.stringify(obj)});
  },
  search = function(query){
    return url("search")+"?"+encSearch(query);
  };

module.exports = {
  url: url,
  search :search,
  thumbSmall: function(id){
    return url("items", id, "thumb/small");
  },
  thumbLarge: function(id){
    return url("items", id, "thumb/large");
  },
  raw: function(id){
    return url("items", id, "raw");
  },
  tags: function(){
    return search({
      "Types":tagTypes,
      "Sort":{ "Created":"desc" }
    });
  },
  tag: function(tag){
    return search({
      "Types": tagTypes,
      "Match":{ "Meta.Slug":tag }
    });
  },
  recent: function(){
    return search({
      "Types": mediaTypes,
      "Sort": {"Created": "desc" }
    });
  },
  item: function(id){
    return url("items", id);
  }
};