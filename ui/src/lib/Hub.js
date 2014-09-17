"use strict";
var EventEmitter = require("events").EventEmitter, _ = require("lodash");
/**
 *  This is a Flux Pattern style Dispatcher and general store.
 *  The ides is that all data is stored in a single object, but
 *  "stores" may manage their data through the hub.
 */
function Hub(){
  this._state = {};
  this._events = new EventEmitter();
  this._stores = {};
  this._dispatching = false;
  this._dispatch = function(){
    if(!this._dispatching){
      process.nextTick(function(){
        this._events.emit("dispatch");
        this._dispatching = false;
      }.bind(this));
    }
    this._dispatching = true;
  };
}


//our global store registry, allows Stores to register themselves (on module load);
//Stores must implement the interface
// new Store(hub) => Store instance
// StoreInstance.dump() => obj
// StoreInstance.inflate(obj)
// the obj must be JSON serialiseable and not lose data in the process.
var StoreRegister = {};

//this is what should be called by store modules to register themselves.
Hub.RegisterStore = function(name, storeConstructor){
  StoreRegister[name] = storeConstructor;
};

Hub.prototype = {
  toJSON: function(){
    var dump = {
      state: this._state,
      stores: {}
    };
    _.each(this._stores, function(v, k){
      dump.stores[k] = v.dump();
    });
    return dump;
  },
  inflate: function(dump){
    this._state = dump.state;
    _.each(dump.stores, function(v,k){
      if(!(k in StoreRegister)){
        throw new Error("Cannot create Store: `"+k+"`");
      }
      //create the store.
      this._stores[k] = StoreRegister[k](this);
      this._stores[k].inflate(v);
    }, this);
    //done
  },
  //get a store instance from the hub.
  getStore: function(name){
    return this._stores[name];
  },
  //register an on-change callback
  register: function(fn){
    this._events.addListener("dispatch", fn);
  },
  //register an event handler (these are the "actions")
  on: function(evt, fn){
    this._events.addListener.call(this._events, "event:"+evt, fn);
  },
  //tell the hub to do something, e.g. trigger an action
  dispatch: function(evt){
    var args = Array.prototype.slice.call(arguments, 1);
    args.unshift("event:"+evt);
    this._events.emit.apply(this._events, args.slice());
    args.unshift("event:any");
    this._events.emit.apply(this._events, args);
  },
  //get a value out of the hub.
  get: function(ref){
    var k = ref, v = this._state;
    if(!k){ return; }
    if(!Array.isArray(k)){
      k = k.replace(/(^\.|\.$)/g, "").split(".");
    }
    while(k.length && (v = v[k.shift()])){}
    return v;
  },
  //definitely private, does the setting, and returns true (change) or false (no change)
  _set: function(ref, val){
    var k = ref, v = this._state, changed = false;
    if(!k){ return changed; }
    if(!Array.isArray(k)){
      k = k.replace(/(^\.|\.$)/g, "").split(".");
    }
    while(k.length >1){
      if(!(k[0] in v)){
        v[k[0]] = {};
      }
      v = v[k.shift()];
    }
    if(val === void 0){
      changed = (v[k[0]] !== void 0);
      delete v[k[0]];
    }else{
      //check before for equality checking, this is not perfect
      //but it's a good indicator
      changed = (JSON.stringify(v[k[0]]) !== JSON.stringify(val));
      v[k[0]] = val;
    }
    //here do a diff!
    return changed;
  },
  //set a value
  set: function(ref, val){
    var change;
    if(typeof ref === "object" && !Array.isArray(ref)){
      //object, set keys => values.
      change = Object.keys(ref).reduce(function(p, c){
        return (this._set(c, ref[c]) || p);
      }.bind(this), false);
    }else{
      change = this._set(ref, val);
    }
    if(change){
      this._dispatch();
    }
    return change;
  },
  //alias setState with undefined value
  unset: function(ref){
    this.set(ref);
  },
  //this should only be used by "privileged" object who manipulate references in
  //the hub directly.
  notify: function(){
    this._dispatch();
  }
};

module.exports = Hub;