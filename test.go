// A test site that creates route requests and shows the result on a map
//
// The polylines of steps are colored alternating in red and blue.

package main

import (
    "html/template"
    "net/http"
)

var testTemplate = template.Must(template.New("main").Parse(testTemplateHTML))

func test(w http.ResponseWriter, r *http.Request) {
    if err := testTemplate.Execute(w, nil); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
    }
}

const testTemplateHTML = `
<!DOCTYPE html>
<html>
<head>
  <title>Team FortyTwo Route Test</title>
  
  <script src="https://ajax.googleapis.com/ajax/libs/jquery/1.7.1/jquery.min.js"></script>
  <script src="http://tile.cloudmade.com/wml/latest/web-maps-lite.js"></script>
  
  <script type="text/javascript">
var map;
function init() {
  map = new CM.Map('map', new CM.Tiles.CloudMade.Web({key: '785cf87085dc4fa08c07a9901126cb49'}));
  map.setCenter(new CM.LatLng(49.25709000000001, 7.045980000000001), 16);
}
</script>

</head>
<body onload="init()">
  <div id="controls" style="height: 60px">
  Parameters: <input id="testParameters" type="text" name="paramters" size="120" value="waypoints=49.2572069321567,7.04588517266191|49.2574019507051,7.04324261219973" />
  <input id="testButton" type="button" name="test" value="Test" />
  </div>
  <div id="map" style="width: 1024px; height: 600px"></div>

<script type="text/javascript">
function routeSuccess(data) {
  var se = data.boundingBox.se;
  var nw = data.boundingBox.nw;
  map.zoomToBounds(new CM.LatLngBounds(new CM.LatLng(se[0], nw[1]) /* sw */, new CM.LatLng(nw[0], se[1]) /* ne */));
    
  var leg = data.routes[0].legs[0];
  $.each(leg.steps, function(i, step){
    var lineColor = "red";
    if (i % 2 ==  0) {
      lineColor = "blue";
    }
    var line = []
    $.each(step.polyline, function(i, point){
      line[i] = new CM.LatLng(point[0], point[1]);
    });
    var polygon = new CM.Polyline(line, lineColor, 5, 0.7);
    map.addOverlay(polygon);
  });
}

function routeError(jqXHR, textStatus, errorThrown) {
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

$(document).ready( function() {
  $("#testButton").click(update);
});
</script>

</body>
</html>
`

