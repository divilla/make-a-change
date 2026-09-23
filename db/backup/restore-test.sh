#!/bin/bash

gunzip < $1 | psql --echo-errors -h localhost -U postgres -X changes_test -F c
