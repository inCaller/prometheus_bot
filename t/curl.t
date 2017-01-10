#!/bin/bash
files=$(find testdata -name *.json)
echo "1..$(echo "$files"|wc -l)"
for file in $files
do
    result="$(curl -sw '\n%{http_code}' --data-binary "@$file" 127.0.0.1:9087/alert/${TELEGRAM_CHATID})"
    code=$(echo "$result"|tail -1)
    responce=$(echo "$result"|head -n -1 )
    test "$code" -eq 200 && msg="ok" || msg="not ok"
    echo "$msg $((++i)) -" ${file#testdata/}
    echo $responce|jq -C . 2>/dev/null| sed -E 's/^(.)/# \1/'
done
