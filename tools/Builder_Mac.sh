#!/bin/zsh
current_dir=$(pwd)
script_dir=$(
    cd $(dirname $0)
    pwd
)

cd $script_dir/../

python3 -m venv .venv
. ./.venv/bin/activate
pip install -r ./s3s/requirements.txt
pip install pyinstaller
pyinstaller --onefile ./s3s/s3s.py

mv ./dist/s3s ./s3s.app
rm -f ./s3s.spec
rm -rf ./build
rm -rf ./dist

cd $current_dir
