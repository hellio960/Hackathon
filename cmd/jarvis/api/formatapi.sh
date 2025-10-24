#!/bin/bash

work_dir=$(cd $(dirname $0); pwd)

repaire_api_file() {
    local file=$1
    local type_name=$2

    # add the type name to the last line
    echo "type ($type_name struct{})" >> $file
    local out=$(goctl api format --dir $file -declare 2>&1 | grep 'not defined')
    if [[ $out =~ ^([a-zA-Z0-9]+)\.api,.*type\ ([a-zA-Z0-9]+)\ not\ defined$ ]]; then
        local type_name=${BASH_REMATCH[2]}
        echo "file: $file, type: $type_name"
        repaire_api_file $file $type_name
    fi

    # check if last line starts with "type"
    local last_line=$(tail -n 1 $file)
    # trim the last line
    last_line=$(echo $last_line | xargs)
    # remove the last line
    sed -i '' '$d' $file
    # if last line starts with }, add it back
    if [[ $last_line =~ ^\} ]]; then
        echo "}" >> $file
    fi

}

process_sub_dir() {
    local dir=$1

    local out=$(goctl api format --dir $dir -declare 2>&1 | grep 'not defined')
    # output: 
    # customer.api, type DiskWideInfo not defined
    # detecter.api, type NodeNicItemReq not defined
    # get the file name and the type name
    while read line; do
        # echo $line
        if [[ $line =~ ^([a-zA-Z0-9]+)\.api,.*type\ ([a-zA-Z0-9]+)\ not\ defined$ ]]; then
            local file_name=${BASH_REMATCH[1]}
            local type_name=${BASH_REMATCH[2]}
            file_path=$dir/$file_name.api
            echo "file: $file_path, type: $type_name"

            # repaire the api file
            repaire_api_file $file_path $type_name
        fi
    done <<< "$out"
}


dirs=$(find $work_dir -type d)
for dir in $dirs; do
    # skip the current dir
    if [[ $dir == $work_dir ]]; then
        continue
    fi
    echo $dir
    process_sub_dir $dir
done