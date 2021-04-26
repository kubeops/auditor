---
title: Auditor Usage
menu:
  docs_{{ .version }}:
    identifier: auditor-usage
    name: Auditor Usage
    parent: reference-operator
menu_name: docs_{{ .version }}
section_menu_id: reference
---
## auditor usage

Generate usage for billing and monitoring

```
auditor usage [flags]
```

### Options

```
      --credential-file string   User credential for connecting to nats server
  -h, --help                     help for usage
      --server string            Nats server endpoint (default "nats://localhost:4222")
      --subject string           Channel name from which events to be listened (default "ClusterEvents")
```

### Options inherited from parent commands

```
      --alsologtostderr                  log to standard error as well as files
      --enable-analytics                 Send analytical events to Google Analytics (default true)
      --log-flush-frequency duration     Maximum number of seconds between log flushes (default 5s)
      --log_backtrace_at traceLocation   when logging hits line file:N, emit a stack trace (default :0)
      --log_dir string                   If non-empty, write log files in this directory
      --logtostderr                      log to standard error instead of files (default true)
      --stderrthreshold severity         logs at or above this threshold go to stderr
      --use-kubeapiserver-fqdn-for-aks   if true, uses kube-apiserver FQDN for AKS cluster to workaround https://github.com/Azure/AKS/issues/522 (default true)
  -v, --v Level                          log level for V logs
      --vmodule moduleSpec               comma-separated list of pattern=N settings for file-filtered logging
```

### SEE ALSO

* [auditor](/docs/reference/operator/auditor.md)	 - Kubernetes Auditor by AppsCode

