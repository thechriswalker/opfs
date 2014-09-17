//The Item Store is an LRU Cache for the items we use
//They have a property item.Hash, which is unique.
//that is our key. and it's hardcoded.
var Hub = require("./Hub");

//The actual LRU implementation
function ItemStore(hub){
  this._hub = hub;
  this.reset();
}

ItemStore.prototype = {
  dump: function(){
    //we can rebuild from this.
    return {
      cap: this._cap,
      data: this._arr
    };
  },
  inflate: function(dump){
    this.reset();
    this._arr = dump.data||[];
    this._arr.forEach(function(i){
      this._map[i.k] = i;
    });
    this.setCapacity(dump.cap||Number.MAX_VALUE);
  },
  reset: function(){
    this._map = {};
    this._arr = [];
    this._cap = Number.MAX_VALUE;
  },
  setCapacity: function(n){
    if(typeof n !== "number" || n < 1){
      throw TypeError("n must be a number greater than 0");
    }
    this._cap = n;
    var item, current = this._arr.length;
    while(current > n){
      item = this._arr.pop();
      delete this._map[item.k];
      current--;
    }
  },
  put: function(key, value){
    var oldest, item = {k:key, v:value};
    this._map[key] = item;
    this._arr.unshift(item);
    if(this._arr.length > this._cap){
      oldest = this._arr.pop();
      delete this._map[oldest.Hash];
    }
    //return the evisted item, if any...
    return oldest;
  },
  get: function(key){
    var idx, item = this._map[key];
    if(!!item){
      //it exists. move it to the head.
      idx = this._arr.indexOf(item);
      this._arr.splice(idx,1); //remove from current position
      this._arr.unshift(item); //put at head.
      return item.v;
    }
  }
};

// this is the important bit.
Hub.RegisterStore("items", function(hub){ return new ItemStore(hub); });

//we don't actually need to do this...
module.exports = ItemStore;