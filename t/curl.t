#!/bin/bash
json_files=$(find testdata -name *.json)
template_files=$(find testdata -name '*.tmpl')

echo "1..$(($(echo "$json_files"|wc -l) * $(printf "$template_files\n\n"|wc -l)))"
echo -n "" > bot.log
for template_file in "" $template_files
do
    echo "****** Run prometheus_bot with template ${template_file} ******" >> bot.log 2>&1
    ./prometheus_bot $(test -n "${template_file}" && echo "-d -t ${template_file}") >> bot.log 2>&1 &
    sleep 3
    for json_file in $json_files
    do
        result="$(curl -sw '\n%{http_code}' --data-binary "@$json_file" 127.0.0.1:9087/alert/${TELEGRAM_CHATID})"
        code=$(echo "$result"|tail -1)
        responce=$(echo "$result"|head -n -1 )
        test "$code" -eq 200 && msg="ok" || msg="not ok"
        echo "$msg $((++i)) -" ${json_file#testdata/} "template" $(test -n "${template_file}" && echo ${template_file#testdata/} || echo "none")
        echo $responce|jq -C . 2>/dev/null| sed -E 's/^(.)/# \1/'
    done
    kill $! 2>/dev/null
    wait $! 2>/dev/null
done
exit 0
