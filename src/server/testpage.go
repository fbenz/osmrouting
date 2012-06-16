// A test site that creates route requests and shows the result on a map
//
// The polylines of steps are colored alternating in red and blue.

package main

import (
	"html/template"
	"net/http"
)

// status returns an HTML page that can be used to test the routing service
func test(w http.ResponseWriter, r *http.Request) {
	if err := testTemplate.Execute(w, nil); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

var testTemplate = template.Must(template.New("test").Parse(testTemplateHTML))

const testTemplateHTML = `
<!DOCTYPE html>
<html>
<head>
  <title>Team FortyTwo Route Test</title>
  
  <script type="text/javascript" src="https://ajax.googleapis.com/ajax/libs/jquery/1.7.1/jquery.min.js"></script>
  <script type="text/javascript" src="http://tile.cloudmade.com/wml/latest/web-maps-lite.js"></script>
  <script type="text/javascript" src="https://maps.googleapis.com/maps/api/js?sensor=false"></script>
  
  <script type="text/javascript">
var oldOverlays = [];
var refOverlays = [];
var map;
var directionsService = new google.maps.DirectionsService();
var floats = [];
var travelmode = travelmode = google.maps.DirectionsTravelMode.DRIVING;

function init() {
  map = new CM.Map('map', new CM.Tiles.CloudMade.Web({key: '785cf87085dc4fa08c07a9901126cb49'}));
  map.setCenter(new CM.LatLng(49.25709000000001, 7.045980000000001), 16);
}
</script>

</head>
<body onload="init()">
  <div id="controls" style="height: 80px">
  Parameters: <input id="testParameters" type="text" name="paramters" size="130" value="waypoints=49.2572069321567,7.04588517266191|49.2574019507051,7.04324261219973&travelmode=walking" />
  <input id="testButton" type="button" name="test" value="Go" style="width: 100px"/>
  <br />
  Reference route: 
  <a href="https://developers.google.com/maps/documentation/javascript/directions">Google Directions</a>
  (support for the first two waypoints and the travel mode)
  <br />
  <input id="testPortCheck" type="checkbox" name="portcheck" value="" />Alternative port: 
  <input id="testPort" type="text" name="port" size="30" value="23401" />
  </div>
  <div id="controls" style="width: 200px; margin-right: 10px; float: left">
    <p id="routeOverview"></p>
    <p id="refOverview"></p>
    <ol id="routeInfo"></ol>
  </div>
  <div id="map" style="width: 800px; height: 600px; float: left"></div>

<script type="text/javascript">
function cleanUp() {
  $.each(oldOverlays, function(i, overlay) {
    map.removeOverlay(overlay);
  });
  oldOverlays = [];

  $("#routeOverview").empty();
  $("#routeInfo").empty();
}

function refCleanUp() {
  $.each(refOverlays, function(i, overlay) {
    map.removeOverlay(overlay);
  });
  refOverlays = [];

  $("#refOverview").empty();
}

function routeSuccess(data) {
  cleanUp();

  var se = data.boundingBox.se;
  var nw = data.boundingBox.nw;
  map.zoomToBounds(new CM.LatLngBounds(new CM.LatLng(se[0], nw[1]) /* sw */, new CM.LatLng(nw[0], se[1]) /* ne */));
  
  $("#routeOverview").append("Total: " + data.routes[0].duration.text + ", " + data.routes[0].distance.text);
    
  var route = data.routes[0];
  $.each(route.legs, function(i, leg){
    $.each(leg.steps, function(i, step){
      $("#routeInfo").append("<li>" + step.instruction + " (" + step.duration.text + ", " + step.distance.text + ")</li>");
  
      var lineColor = "red";
      if (i % 2 ==  0) {
        lineColor = "blue";
      }
      if (i == 0) {
      	lineColor = "olive"
      }
      var line = []
      $.each(step.polyline, function(i, point) {
        line[i] = new CM.LatLng(point[0], point[1]);
      });
      var polygon = new CM.Polyline(line, lineColor, 5, 0.7);
      map.addOverlay(polygon);
      oldOverlays.push(polygon);
    });
  });
  
  var start = new CM.LatLng(floats[0], floats[1]);
  var startMarker = new CM.Marker(start);
  var end = new CM.LatLng(floats[2], floats[3]);
  var endMarker = new CM.Marker(end);
  map.addOverlay(startMarker);
  oldOverlays.push(startMarker);
  map.addOverlay(endMarker);
  oldOverlays.push(endMarker);
}

function routeError(jqXHR, textStatus, errorThrown) {
  cleanUp();

  var ts = "";
  if (textStatus != null) {
    ts = textStatus;
  }
  var et = "";
  if (errorThrown != null) {
    et = errorThrown;
  }
  var rt = "";
  if (jqXHR != null && jqXHR.responseText != null) {
    rt = jqXHR.responseText;
  }
  alert(ts + ": " + et + ": " + rt);
}

function getParam(params, name) {
  var ret = null;
  $.each(params, function(i, param){
  	if (param.indexOf(name) >= 0) {
      var j = param.indexOf(name);
      var start = j + name.length;
      ret = param.substring(start, param.length);
      return false;
  	}
  });
  return ret;
}

function extractWaypoints(w) {
	var points = w.split("|");
	floats = [];
	$.each(points, function(i, point){
		var p = point.split(",");
		floats[2*i] = parseFloat(p[0]);
		floats[2*i+1] = parseFloat(p[1]);
	});
}

function extractTravelmode(mode) {
	travelmode = google.maps.DirectionsTravelMode.DRIVING;
	if (mode == null) {
		return;
	}
	mode = mode.toLowerCase();
	if (mode == "walking") {
		travelmode = google.maps.DirectionsTravelMode.WALKING;
	} else if (mode == "bicycling") {
		travelmode = google.maps.DirectionsTravelMode.BICYCLING;
	}
}

function update() {
  var urlParam = $("#testParameters").val();
  var params = urlParam.split("&");
  extractWaypoints(getParam(params, "waypoints="));
  extractTravelmode(getParam(params, "travelmode="));
  
  var url = "/routes?" + $("#testParameters").val();
  if ($("#testPortCheck").is(':checked')) {
  	 url = "/forward?port=" + $("#testPort").val() + "&" + $("#testParameters").val()
  };

  $.ajax({
    url: url,
    dataType: 'json',
    success: routeSuccess,
    error: routeError
  });
  refUpdate();
}

function refUpdate() {
  refCleanUp();

  var request = {
    origin: new google.maps.LatLng(floats[0], floats[1]), 
    destination: new google.maps.LatLng(floats[2], floats[3]),
    travelMode: travelmode,
    unitSystem: google.maps.UnitSystem.METRIC
  };
  directionsService.route(request, function(response, status) {
    if (status == google.maps.DirectionsStatus.OK) {
      $("#refOverview").append("Reference: " + response.routes[0].legs[0].duration.text + ", " + response.routes[0].legs[0].distance.text);
      
      var leg = response.routes[0].legs[0];
      $.each(leg.steps, function(i, step){
        var polyline = google.maps.geometry.encoding.decodePath(step.polyline.points);
      
        var line = []
        $.each(polyline, function(i, point) {
          line[i] = new CM.LatLng(point.lat(), point.lng());
        });
        var polygon = new CM.Polyline(line, "green", 5, 0.5);
        map.addOverlay(polygon);
        refOverlays.push(polygon);
      });
    }
  });
}

$(document).ready( function() {
  $("#testButton").click(update);
  $("#refButton").click(refUpdate);
  
  $("#testParameters").keyup(function(event) {
    if(event.keyCode == 13){
      $("#testButton").click();
    }
  });
  $("#refParameters").keyup(function(event) {
    if(event.keyCode == 13){
      $("#refButton").click();
    }
  });
});
</script>

</body>
</html>
`
