OSM Routing
==========

A fast and optimal routing service for cars, bikes, and pedestrians that is based on OpenStreetMap data. The project consists of a server and several programs for the preprocessing (parser, refine, partition, metric, kdtreebuilder). The service scales up to the whole planet meaning that the preprocessing takes a few hours and queries on >1000km routes take less than a second (assuming a modern computer).

Authors
---------
Florian Benz,
Steven Schäfer,
Bernhard Schommer

Requirements
---------
* Go version 1.0 or above (tested with 1.0 and 1.2)
* Metis: 'gpmetis' has to be available for the preprocessing

Running
-------------

To build everything, use:

    build.sh

To execute the preprocessing steps, use:

    preprocess.sh pbf_file path

To start up the server, use:

    bin/server -dir path -port port

The following parameters occur in the commands above:
* pbf_file: OSM PBF file
* path: absolut path to the graph dir
* port: port of the server

Background
-------------

This project started during the Algorithm Engineering course at Saarland University in summer 2012 (http://www.mpi-inf.mpg.de/departments/d1/teaching/ss12/alg_eng/). In a team of three we successfully completed the course project where the task was to create a fast routing service based on OSM data. The main algorithm is based on ideas from the following paper:

D. Delling, A. V. Goldberg, T. Pajor, R. F. Werneck, Customizable route planning. SEA'11. 
