//The Selection Store holds the current selection. (and the items)
var _ = require("lodash"),
    Hub = require("./Hub");

function SelectionStore(hub){
  this._hub = hub;
  this.reset();
}

_.extend(SelectionStore.prototype,{
  reset: function(){
    this._selection = [];
    this._map = {};
  },
  dump: function(){
    return { data: this._map };
  },
  inflate: function(dump){
    this._map = dump.data||{};
    this._selection = Object.keys(this._map);
  },
  _getItem: function(id){
    return this._hub.getStore("items").get(id);
  },
  contains: function(id){
    return (id in this._map);
  },
  selected: function(){
    return this.selection.slice();
  },
  addItem: function(id){
    if(this.contains(id)){ return; }
    var item = this._getItem(id);
    if(!item){ return; }
    this._selection.push(id);
    this._map[id] = item;
    this._hub.notify();
  },
  removeItem: function(id){
    if(!this.contains(id)){ return; }
    this._selection.splice(this._selection.indexOf(id), 1);
    delete this._map[id];
    this._hub.notify();
  },
  getItem: function(id){
    return this._map[id];
  },
  toggleItem: function(id){
    if(this.contains(id)){
      this.removeItem(id);
    }else{
      this.addItem(id);
    }
  }
});

// this is the important bit.
Hub.RegisterStore("selection", function(hub){ return new SelectionStore(hub); });

//we don't actually need to do this...
module.exports = SelectionStore;