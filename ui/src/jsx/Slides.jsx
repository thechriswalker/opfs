/** @jsx React.DOM */
var React = require("react"),
    SlideComponents = require("./SlideComponents");

module.exports = React.createClass({
  displayName: "Slides",
  render: function(){
    var hub = this.props.hub, fragment = hub.get("url.fragment");

    //if fragment matches!
    var matches = fragment && fragment.match(/^slideshow\/(sha1-[a-z0-9]{40})$/);
    if(!matches){
      //no show, show no.
      return null;
    }
    //we have one! get the id's of prev and next.
    //We might want to slideshow a specific selection of images, so we
    //might need an overload for "selection" vs "loaded".
    var itemId = matches[1],
        set = hub.get("items"),
        itemIdx = set.indexOf(itemId),
        prev = itemIdx > 0 ? set[itemIdx-1] : false,
        next = itemIdx < set.length-1 ? set[itemIdx+1] : false,
        item = hub.getStore("items").get(itemId);

    if(itemIdx < 0 || !item){
      //not in items, or not available!
      return null;
    }

    return <SlideComponents.Overlay>
      <SlideComponents.ItemView item={item} />
      <SlideComponents.DetailView item={item} />
      <SlideComponents.PrevLink id={prev} />
      <SlideComponents.NextLink id={next} />
    </SlideComponents.Overlay>;
  }
});
