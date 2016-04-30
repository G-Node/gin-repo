#!/bin/bash
NAME=${1:-exrepo}

git init "$NAME"
pushd "$NAME"

git config user.name "gin repo"
git config user.email "gin-repo@g-node.org"

curl -o code.zip https://portal.g-node.org/doi/10.12751/g-node.t6vbz9/kellner2016_colortilt_code.zip

unzip -qq code.zip
mv colortilt-josaa_2016a/* . && rm -r colortilt-josaa_2016a
git add .
git commit -m "Experiment sources added"

# non-broken symlink example
ln -s analysis/paper.sh paper.sh

git annex init

curl -o data.zip https://portal.g-node.org/doi/10.12751/g-node.t6vbz9/kellner2016_colortilt_data.zip
git annex add data.zip

git commit -m "Add data.zip (annexed)"

# make a tag, that has a slash in it
git tag -a -m "Tag as paper/jossa" paper/jossa

# make a topic branch and merge it with --no-ff
git checkout -b topic
git add paper.sh
git commit -m "Add paper.sh script"

git checkout master
git merge -m "merge topic branch" --no-ff topic
git branch -d topic

popd

git clone --bare "$NAME" "$NAME.git"
git --git-dir="$NAME.git" annex init

pushd "$NAME"
git remote add local ../"$NAME.git"
git push local master
git annex sync local --content

popd
