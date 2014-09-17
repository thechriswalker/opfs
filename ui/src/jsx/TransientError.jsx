/** @jsx React.DOM */
var React = require("react");

/**
 *  Helper for transient errors for things that can be retried.
 *  The retry function should do something that unmounts this component, or
 *  if the same state happens again, this component will stay mounted and therefore
 *  not reset.
 */
module.exports = React.createClass({
  displayName: "TransientError",
  propTypes: {
    retry: React.PropTypes.func.isRequired,
    timer: React.PropTypes.number.isRequired
  },
  render: function(){
    var timeLeft = this.state.timeLeft > 0,
        subText = this.props.retryNowText || "retrying now...",
        buttonText = this.props.buttonText || "retry now";
    if(timeLeft){
      subText = (this.props.subText || "retrying in {:seconds}s").replace("{:seconds}", this.state.timeLeft);
    }
    return <div className="alert alert-danger">
      {timeLeft && <button className="btn btn-default pull-right btn-xs" onClick={this.doNow}>{buttonText}</button>}
      <strong>{this.props.main}</strong>
      {" "}
      {subText}
    </div>;
  },
  tick: function(){
    if(!this.isMounted()){
      //something went wrong. handle it gracefully
      clearInterval(this.state.interval);
      return;
    }
    var left = this.state.timeLeft - 1;
    if(left === 0){
      this.doNow();
    }
    this.setState({
      timeLeft: left
    });
  },
  doNow: function(){
    //call the passed in function.
    process.nextTick(this.props.retry);
    //clear the interval
    clearInterval(this.state.interval);
    this.setState({timeLeft: 0});
  },
  getInitialState: function(){
    return {
      timeLeft: Math.floor(this.props.timer)
    };
  },
  componentDidMount: function(){
    this.setState({
      interval: setInterval(this.tick, 1e3)
    });
  },
  componentWillUnmount: function(){
    clearInterval(this.state.interval);
  }
});