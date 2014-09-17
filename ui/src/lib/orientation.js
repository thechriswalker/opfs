/*
  To help with EXIF Orientation Flags.
*/

//radian helpers...
var deg90 = Math.PI/2,
    deg180 = Math.PI,
    deg270 = Math.PI*3/2;

var ORIENTATIONS = [
  { name: "Undefined", mirrored:false, rotation: 0 },
  { name: "Normal", mirrored:false, rotation: 0 },
  { name: "Mirror", mirrored:true, rotation: 0 },
  { name: "Normal180", mirrored:false, rotation: deg180 },
  { name: "Mirror180", mirrored:true, rotation:  deg180},
  { name: "Mirror270", mirrored:true, rotation: deg270 },
  { name: "Normal270", mirrored:false, rotation: deg270 },
  { name: "Mirror90", mirrored:true, rotation: deg90 },
  { name: "Normal90", mirrored:false, rotation: deg90 }
];

ORIENTATIONS.isMirrored = function(flag){
  return ORIENTATIONS[flag] && ORIENTATIONS[flag].mirrored;
};

ORIENTATIONS.getRotation = function(flag){
  return ORIENTATIONS[flag] && ORIENTATIONS[flag].rotation;
};

ORIENTATIONS.getName = function(flag){
  return ORIENTATIONS[flag] && ORIENTATIONS[flag].name;
};

//get the CSS transform matrix.
ORIENTATIONS.getTransform = function(flag){
  var angle = ORIENTATIONS.getRotation(flag),
      mirror = ORIENTATIONS.isMirrored(flag),
      /*
      matrix is: ( a b tx )
                 ( c d ty )
                 ( 0 0 1  )

      argument order in css: a, c, b, d, tx, ty
      */
      matrix = [
        Math.cos(angle)*(mirror?-1:1), //a
        Math.sin(angle), //b
        -Math.sin(angle), //c
        Math.cos(angle), //d
        0, //tx
        0, //ty
      ].map(function(n){
        return ""+Math.round(n);
      }).join(",");

      return "matrix("+matrix+");";
};

var style = "";
ORIENTATIONS.forEach(function(o, i){
  style += ".exif-orientation-"+i+" {\n"+
    "  transform: "+ORIENTATIONS.getTransform(i)+"\n"+
    "}\n";
});

console.log(style);