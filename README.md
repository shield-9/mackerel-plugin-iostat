# mackerel-plugin-iostat

## Install
```bash
mkr plugin install mackerel-plugin-iostat --upgrade
```

## Configuration
```TOML
[plugin.metrics.iostat]
command = "/opt/mackerel-agent/plugins/bin/mackerel-plugin-iostat --ignore-virtual"
```

