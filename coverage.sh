#!/bin/bash
echo "mode: count" > profile.cov;
for Dir in $(go list ./...);
do
  currDir=$(echo $Dir | sed 's/github.com\/G-Node\/gin-repo\///g');
  currCount=$(ls -l $currDir/*.go 2>/dev/null | wc -l);
  if [ $currCount -gt 0 ];
  then
    go test -covermode=count -coverprofile=tmp.out $Dir;
    if [ -f tmp.out ];
    then
      cat tmp.out | grep -v "mode: count" >> profile.cov;
    fi
  fi
done

