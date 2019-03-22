# MultiTenancies
[![All Contributors](https://img.shields.io/badge/all_contributors-2-orange.svg?style=flat-square)](#contributors)
`MultiTenancy` is a kubernetes workload type like `StatefulSet`, except dependant on defined tenants instead of on a replica count.

## Installation

```
kubectl apply -f https://raw.githubusercontent.com/configurator/multitenancy/master/deployment/multitenancy.yaml
```

## Example

And you have a few items deployed:

```yaml
apiVersion: confi.gurator.com/v1
kind: Tenant
tenancyKind: food
metadata:
  name: apple
data:
  name: Apple
  type: Fruit

---

apiVersion: confi.gurator.com/v1
kind: Tenant
tenancyKind: food
metadata:
  name: cucumber
data:
  name: Cucumber
  type: Vegetable

---

apiVersion: confi.gurator.com/v1
kind: Tenant
tenancyKind: food
metadata:
  name: tomato
data:
  name: Tomato
  type: Fruit
```

`MultiTenancies` allow you to create a pod for each of those items. Defining a `MultiTenancy` is much like defining a `Deployment`, or a `StatefulSet`, expect instead of adding a replica count, we tell the operator which tenancyKind to replicate for
```yaml
apiVersion: confi.gurator.com/v1
kind: MultiTenancy
metadata:
  name: food-processor
spec:
  # We want one instace for each Food in our system (required)
  tenancyKind: food
  # Name the env variable for where to put the name of the food (optional)
  tenantNameVariable: FOOD_TYPE
  # Name the volume for where to put the food details (optional)
  tenantResourceVolume: which-food
  # Note: if the tenantResourceVolume is specified, the pod will be restarted for any change in the tenant's data.
  # However, if only tenantNameVariable is specified, the pod will not respond to changes in tenant data

  # Standard selector and template definition, much like for Deployments or StatefulSets:
  selector:
    matchLabels:
      app: food-processor
  template:
    metadata:
      labels:
        app: food-processor
    spec:
      containers:
      - name: food-processor
        image: my-food-processor
        # Add a volume so we can know which Food this pod belongs to
        volumes:
          - name: which-food
            mountPath: /etc/which-food
```

The `MultiTenancy` operator will now create a Pod for each `Tenant` in the system with `tenancyKind: food`, and mount a volume on each one at `/etc/which-food`. Inside that directory, we'll have two files, named `name` and `type`, containing the data from the `Food` definition, much like it would if we had a `ConfigMap`.

If a `Food` item is created or deleted, the pods are killed and started appropriately; for updates, the pod is first killed, then a new one is started.

## Contributors

Thanks goes to these wonderful people ([emoji key](https://allcontributors.org/docs/en/emoji-key)):

<!-- ALL-CONTRIBUTORS-LIST:START - Do not remove or modify this section -->
<!-- prettier-ignore -->
<table><tr><td align="center"><a href="http://confi.gurator.com"><img src="https://avatars3.githubusercontent.com/u/671365?v=4" width="100px;" alt="Dor Kleiman"/><br /><sub><b>Dor Kleiman</b></sub></a><br /><a href="#ideas-configurator" title="Ideas, Planning, & Feedback">🤔</a> <a href="https://github.com/configurator/multitenancy/commits?author=configurator" title="Code">💻</a></td><td align="center"><a href="http://www.ronin.co.il"><img src="https://avatars2.githubusercontent.com/u/846044?v=4" width="100px;" alt="Alon Valadji"/><br /><sub><b>Alon Valadji</b></sub></a><br /><a href="#ideas-alonronin" title="Ideas, Planning, & Feedback">🤔</a> <a href="https://github.com/configurator/multitenancy/commits?author=alonronin" title="Documentation">📖</a> <a href="#content-alonronin" title="Content">🖋</a></td></tr></table>

<!-- ALL-CONTRIBUTORS-LIST:END -->

This project follows the [all-contributors](https://github.com/all-contributors/all-contributors) specification. Contributions of any kind welcome!