# set http proxy in wsl2
set_proxy() {
    local hostip=$(cat /etc/resolv.conf | grep nameserver | awk '{ print $2 }')
    local proxy_url="http://${hostip}:7890"
    export https_proxy="$proxy_url"
    export http_proxy="$proxy_url"
    export all_proxy="$proxy_url"
}

unset_proxy() {
    export https_proxy="" http_proxy="" all_proxy=""
}
