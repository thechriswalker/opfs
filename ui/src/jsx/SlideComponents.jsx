/** @jsx React.DOM */
var React = require("react"),
    cx = require("react/lib/cx"),
    apiUrl = require("../lib/api_urls");

//the dark overlay to prevent click-through
var Overlay = React.createClass({
  displayName: "SlideComponents.Overlay",
  render: function(){
    return <div className="slideshow-overlay">{this.props.children}</div>;
  }
});

//the item it self, i.e. a big picture.
//and I had to use a <table> to vertically align it... :facepalm:
var ItemView = React.createClass({
  displayName: "SlideComponents.ItemView",
  render: function(){
    var itemComponent;
    if(this.props.item.Type === "Video"){
      //we need a video tag and the raw url.

      //might work better without the poster (thumb is smaller than video and video is dynamically resized...)
      //poster={apiUrl.thumbLarge(this.props.item.Hash)}
      //
      itemComponent = <video autoPlay controls src={apiUrl.raw(this.props.item.Hash)} className="slideshow-itemview-item" />;
    }else{
      itemComponent = <img key={this.props.item.Hash} src={apiUrl.raw(this.props.item.Hash)} className="slideshow-itemview-item" />;
    }

    return <div className="slideshow-itemview">
      <div className="slideshow-itemview-wrap">
        {itemComponent}
        <div className="slideshow-itemview-loader">
          <span className="fa-stack">
            <i className="fa fa-circle-o-notch fa-spin fa-stack-2x fa-fw"></i>
            <i className={"fa fa-stack-1x fa-fw opfs-icon-"+this.props.item.Type}></i>
          </span>
        </div>
      </div>
    </div>;
  }
});

//the detail view, i.e. meta-data
var DetailView = React.createClass({
  displayName: "SlideComponents.DetailView",
  render: function(){
    return <div className="slideshow-detailview">
      {"DetailView: "+this.props.item.Hash}
    </div>;
  }
});

//these are the big links to previous and next slideshow images.
var Link = React.createClass({
  displayName: "SlideComponents.Link",
  render: function(){
    var dir = this.props.dir,
        id = this.props.id,
        className = cx({
          "slideshow-link-icon": true,
          "fa" :true,
          "fa-chevron-left": (dir === "prev"),
          "fa-chevron-right": (dir === "next")
        });

    if(!id){ return null; }
    //we should include an unseen img tag for preloading.

    return <a href={"#slideshow/"+id} className={"slideshow-link slideshow-"+dir}>
      <i className={className} />
      <img className="preloader" src={apiUrl.thumbLarge(id)} />
    </a>;
  }
});

var NextLink = React.createClass({
  displayName: "SlideComponents.NextLink",
  render: function(){
    return <Link dir="next" id={this.props.id} />;
  }
});

var PrevLink = React.createClass({
  displayName: "SlideComponents.PrevLink",
  render: function(){
    return <Link dir="prev" id={this.props.id} />;
  }
});


module.exports = {
  Overlay: Overlay,
  ItemView: ItemView,
  DetailView: DetailView,
  NextLink: NextLink,
  PrevLink: PrevLink
};