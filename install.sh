#!/bin/sh
govendor sync &&
for dir in `find plugin/* -type d`; do
  `cd "$dir" && go install`
done
