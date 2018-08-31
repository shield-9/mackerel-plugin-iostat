# mackerel-plugin-iostat

## Install
```bash
mkr plugin install https://cdn.rawgit.com/shield-9/c1f45573620b8163597920592718582e/raw/83316028241941af0dc4ba4f482d74145946da3a/mackerel-plugin-iostat_linux_amd64.zip --overwrite
```

## Configuration
```TOML
[plugin.metrics.iostat]
command = "/opt/mackerel-agent/plugins/bin/mackerel-plugin-iostat --ignore-virtual"
```

