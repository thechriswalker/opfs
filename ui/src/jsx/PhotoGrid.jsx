/** @jsx React.DOM */
var React = require("react"),
    apiUrls = require("../lib/api_urls");


module.exports = React.createClass({
  displayName: "PhotoGrid",
  render: function(){
    var hub = this.props.hub,
        items = this.props.items,
        itemStore = hub.getStore("items"),
        selection = hub.getStore("selection");

    if(!Array.isArray(items)){
      //error, unexpected...
      return null;
    }
    //array, all good.
    if(!items.length){
      return <div className="jumbo inset">no results!</div>;
    }

    //something to show!
    return <div className="photo-grid">
      {items.map(function(hash, i){
        //@TODO this need to be a show full view with detail/large thumb/tagging/album/content options
        var item = itemStore.get(hash),
            selected = selection.contains(hash),
            details = JSON.stringify(item, null, "    ");

        return <div className="photo-grid-thumb" key={i+hash}>
            <input type="checkbox" className="photo-grid-selection-checkbox" onChange={this.handleClick.bind(this, hash)} checked={selected} />
            <a href={"#slideshow/"+hash} className={"photo-grid-item"+(selected ? " photo-grid-item-selected":"")}>
              <div className="thumb-wrap">
                <img title={details} className="img-rounded thumb" src={apiUrls.thumbSmall(hash)} />
                <div className="thumb-placeholder"><i className={"fa thumb-icon-"+item.Type} /></div>
              </div>
            </a>
          </div>;
      }, this)}
    </div>;
  },
  handleClick: function(hash){
    this.props.hub.dispatch("toggle:selection", hash);
  }
});