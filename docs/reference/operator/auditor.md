---
title: Auditor
menu:
  docs_{{ .version }}:
    identifier: auditor
    name: Auditor
    parent: reference-operator
    weight: 0

menu_name: docs_{{ .version }}
section_menu_id: reference
url: /docs/{{ .version }}/reference/operator/
aliases:
- /docs/{{ .version }}/reference/operator/auditor/
---
## auditor

Kubernetes Auditor by AppsCode

### Options

```
  -h, --help                             help for auditor
      --use-kubeapiserver-fqdn-for-aks   if true, uses kube-apiserver FQDN for AKS cluster to workaround https://github.com/Azure/AKS/issues/522 (default true)
```

### SEE ALSO

* [auditor run](/docs/reference/operator/auditor_run.md)	 - Launch Audit operator
* [auditor version](/docs/reference/operator/auditor_version.md)	 - Prints binary version number.

