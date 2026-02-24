{...}: {

  /*
    Trans from 
    ```
    DEVICE=eth0
    BOOTPROTO=static
    ONBOOT=yes
    IPADDR=142.171.239.148
    NETMASK=255.255.255.128
    GATEWAY=142.171.239.129
    IPV6INIT=yes
    IPV6_AUTOCONF=no
    IPV6ADDR=2607:f130:0000:0159:0000:0000:359e:1e17/64
    IPV6_DEFAULTGW=2607:f130:0000:0159::1
    IPV6ADDR_SECONDARIES="2607:f130:0000:0159:0000:0000:287a:3a95/64  2607:f130:0000:0159:0000:0000:ab55:d045/64"
    ```
  */

  systemd.network.enable = true;

  systemd.network.networks."10-eth0" = {
    matchConfig.Name = "eth0";
    address = [
      "142.171.239.148/25"
      "2607:f130:0000:0159::359e:1e17/64"
      "2607:f130:0000:0159::287a:3a95/64"
      "2607:f130:0000:0159::ab55:d045/64"
    ];
    gateway = [
      "142.171.239.129"
      "2607:f130:0000:0159::1"
    ];
    # routes = [
    #   { Gateway = "142.171.239.129"; }
    #   { Gateway = "2607:f130:0000:0159::1"; }
    # ];
    linkConfig.RequiredForOnline = "routable";
  };
}
