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
function init() {
  map = new CM.Map('map', new CM.Tiles.CloudMade.Web({key: '785cf87085dc4fa08c07a9901126cb49'}));
  map.setCenter(new CM.LatLng(49.25709000000001, 7.045980000000001), 16);
}
</script>

</head>
<body onload="init()">
  <div id="controls" style="height: 60px">
  Parameters: <input id="testParameters" type="text" name="paramters" size="130" value="waypoints=49.2572069321567,7.04588517266191|49.2574019507051,7.04324261219973" />
  <input id="testButton" type="button" name="test" value="Go" style="width: 100px"/>
  <br />
  <a href="https://developers.google.com/maps/documentation/javascript/directions">Google Directions</a>: <input id="refParameters" type="text" name="paramters" size="122" value="not working yet..." />
  <input id="refButton" type="button" name="test" value="Get reference" style="width: 100px"/>
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
    
  var leg = data.routes[0].legs[0];
  $.each(leg.steps, function(i, step){
    $("#routeInfo").append("<li>" + step.instruction + " (" + step.duration.text + ", " + step.distance.text + ")</li>");
  
    var lineColor = "red";
    if (i % 2 ==  0) {
      lineColor = "blue";
    }
    var line = []
    $.each(step.polyline, function(i, point) {
      line[i] = new CM.LatLng(point[0], point[1]);
    });
    var polygon = new CM.Polyline(line, lineColor, 5, 0.7);
    map.addOverlay(polygon);
    oldOverlays.push(polygon);
  });
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

function update() {
  $.ajax({
    url: "/routes?" + $("#testParameters").val(),
    dataType: 'json',
    success: routeSuccess,
    error: routeError
  });
}

function refUpdate() {
  refCleanUp();
  var request = {
    origin: new google.maps.LatLng(49.2572069321567, 7.04588517266191), 
    destination: new google.maps.LatLng(49.2574019507051, 7.04324261219973),
    travelMode: google.maps.DirectionsTravelMode.DRIVING,
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
        var polygon = new CM.Polyline(line, "green", 5, 0.7);
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
