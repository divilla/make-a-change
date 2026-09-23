#!/bin/bash

gunzip < $1 | psql --echo-errors -h localhost -U postgres -X changes -F c
