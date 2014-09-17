"use strict";
/* jshint browser:true */
var Promise = require("es6-promise").Promise;

module.exports = {
  get: function(url){
    return xhr("GET",url).then(OK);
  },
  post: function(url, data){
    return xhr("POST", url, data).then(OK);
  },
  put: function(url, data){
    return xhr("POST", url, data).then(OK);
  },
  delete: function(url){
    return xhr("DELETE", url).then(OK);
  },
  //and the lower-level functions
  xhr: xhr,
  OK: OK
};

//nice little promise wrapper for filtering non-2xx responses.
function OK(res){
  if(!res.ok){
    throw Error("Bad Status Code: "+res.status);
  }
  return res;
}

//this is the main function.
function xhr(method, url, data){
  return new Promise(function(resolve, reject){
    var x = new XMLHttpRequest();
    x.open(method, url, true);
    x.addEventListener("load", function(){
      var response = {
        ok: Math.floor(x.status / 100) === 2,
        status: x.status,
        data: null
      };
      if(x.responseText !== ""){
        //otherwise, no response. thats OK. probably a 201/204
        response.data = JSON.parse(x.responseText);
      }
      resolve(response);
    });
    x.addEventListener("error", reject);
    if(data){
      if(typeof data !== "string"){
        data = JSON.stringify(data);
      }
      x.setRequestHeader("content-type", "application/json");
    }
    x.send(data);
  });
}