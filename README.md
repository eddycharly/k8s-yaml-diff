# k8s-yaml-diff

A simple tools to compare two files containing k8s resources.

## Install

Download the release that suits your needs from the [releases page](https://github.com/eddycharly/k8s-yaml-diff/releases).

## Invoke

Mandatory arguments:
- `source` the path to the source file containing k8s manifests
- `target` the path to the target file containing k8s manifests

Optional arguments:
- `mode` should be either `full` (default) or `diff`. In `diff` mode, report will contain only resources that change.
- `normalize` when specified, yaml objects are normalized once loaded.

Exemple:

```shell
k8s-yaml-diff --mode full --source ./test/source.yaml --target ./test/target.yaml
```

It will output a simple report in `markdown` format as shown below.

---

# Changes report [full] [^legend]

| Kind (Version) | Namespace | Name | B | A | :eyes: |
|---|---|---|:-:|:-:|---|
| Namespace (v1) |  | `dev` | :white_check_mark: | :white_check_mark: |  |
| Namespace (v1) |  | `only-in-source` | :white_check_mark: | :red_circle: | [^v1--namespace--only-in-source] |
| Namespace (v1) |  | `only-in-target` | :red_circle: | :white_check_mark: | [^v1--namespace--only-in-target] |
| Namespace (v1) |  | `prod` | :white_check_mark: | :white_check_mark: |  |
| Namespace (v1) |  | `test` | :white_check_mark: | :white_check_mark: | [^v1--namespace--test] |
| RBACDefinition (v1beta1) |  | `rbac` | :white_check_mark: | :white_check_mark: | [^rbacmanagerreactiveopsio--v1beta1--rbacdefinition--rbac] |

[^legend]:
    ### Legend
    - **B :** *Before*
    - **A :** *After*
[^v1--namespace--only-in-source]:
    ### v1 / Namespace / only-in-source
    ```diff
    --- source
    +++ target
    @@ -1,4 +1 @@
    -apiVersion: v1
    -kind: Namespace
    -metadata:
    -  name: only-in-source
    ```
[^v1--namespace--only-in-target]:
    ### v1 / Namespace / only-in-target
    ```diff
    --- source
    +++ target
    @@ -1 +1,4 @@
    +apiVersion: v1
    +kind: Namespace
    +metadata:
    +  name: only-in-target
    ```
[^v1--namespace--test]:
    ### v1 / Namespace / test
    ```diff
    --- source
    +++ target
    @@ -3,4 +3,4 @@
     metadata:
       name: test
       labels:
    -    origin: source
    +    origin: target
    ```
[^rbacmanagerreactiveopsio--v1beta1--rbacdefinition--rbac]:
    ### rbacmanager.reactiveops.io / v1beta1 / RBACDefinition / rbac
    ```diff
    --- source
    +++ target
    @@ -8,4 +8,4 @@
           - kind: User
             name: eddycharly
         clusterRoleBindings:
    -      - clusterRole: view
    +      - clusterRole: edit
    ```
