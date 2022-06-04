---
title: Auditor Version
menu:
  docs_{{ .version }}:
    identifier: auditor-version
    name: Auditor Version
    parent: reference-operator
menu_name: docs_{{ .version }}
section_menu_id: reference
---
## auditor version

Prints binary version number.

```
auditor version [flags]
```

### Options

```
      --check string   Check version constraint
  -h, --help           help for version
      --short          Print just the version number.
```

### Options inherited from parent commands

```
      --use-kubeapiserver-fqdn-for-aks   if true, uses kube-apiserver FQDN for AKS cluster to workaround https://github.com/Azure/AKS/issues/522 (default true)
```

### SEE ALSO

* [auditor](/docs/reference/operator/auditor.md)	 - Kubernetes Auditor by AppsCode

