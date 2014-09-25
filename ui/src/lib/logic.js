//this module exposes the core app logic.
var xhr = require("./json_xhr");

module.exports = {
  Init: function(hub){
    //When the route is called we need to change the app to react to a change in url.
    hub.on("route", function(url){
      hub.set({url: url});
    });

    //this initiates a new search
    hub.on("search", function(url){
      reset(hub);
      xhr.get(url).then(function(res){
        var update = {
          paging: {
            total: res.data.Total,
            last: url,
            next: res.data.Next,
            error: null
          },
          items: []
        },
        itemStore = hub.getStore("items");

        update.items = res.data.Results.map(function(r){
          itemStore.put(r.Hash, r);
          return r.Hash;
        });
        hub.set(update);
      }).catch(function(err){
        //doe something with err.
        console.error(err);
        hub.set("paging.error", err.message);
      });
    });

    //fetch the next page of results.
    hub.on("next", function(){
      var url = hub.get("paging.next");
      if(!url){
        console.warn("`next` dispatched, but no next url!");
        return;
      }
      xhr.get(url).then(function(res){
        var update = {
          paging: {
            total: res.data.Total,
            last: url,
            next: res.data.Next,
          },
          items: hub.get("items"),
        },
        itemStore = hub.getStore("items");

        res.data.Results.forEach(function(r){
          itemStore.put(r.Hash, r);
          update.items.push(r.Hash);
        });

        hub.set(update);
      }).catch(function(err){
        //error at this point... maybe we had better put this elsewhere in the state, so we don't have to zap everything...
        console.error(err);
        hub.set("paging.error", err.message);
      });
    });

    hub.on("toggle:selection", function(hash){
      hub.getStore("selection").toggleItem(hash);
    });
  }
};

function reset(hub){
  hub.getStore("items").reset();
  hub.set({
    paging:{},
    items:null
  });
}