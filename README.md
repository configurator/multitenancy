# ResourcefulSets
`ResourcefulSet` is a kubernetes workload type like `StatefulSet`, except dependant on resources instead of a replica count.

## Installation

```
kubectl apply -f https://raw.githubusercontent.com/configurator/resourceful-set/master/deployment/resourceful-set.yaml
```

## Example

Suppose you have a custom resource type for your application:

```yaml
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: foods.examples.confi.gurator.com
spec:
  group: examples.confi.gurator.com
  versions:
    - name: v1
      served: true
      storage: true
  scope: Namespaced
  names:
    plural: foods
    singular: food
    kind: Food
```

And you have a few items deployed:

```yaml
apiVersion: examples.confi.gurator.com/v1
kind: Food
metadata:
  name: apple
data:
  name: Apple
  type: Fruit
---
apiVersion: examples.confi.gurator.com/v1
kind: Food
metadata:
  name: cucumber
data:
  name: Cucumber
  type: Vegetable
---
apiVersion: examples.confi.gurator.com/v1
kind: Food
metadata:
  name: tomato
data:
  name: Tomato
  type: Fruit
```

`ResourcefulSets` allow you to create a pod for each of those items. Defining a `ResourcefulSet` is much like defining a `Deployment`, or a `StatefulSet`, expect instead of adding a replica count, we tell the operator what to replicate based on:
```yaml
apiVersion: confi.gurator.com/v1
kind: ResourcefulSet
metadata:
  name: food-processor
spec:
  # We want one instace for each Food in our system
  replicateForResource: Food
  # Name the volume for where to put the food details
  replicationResourceVolume: which-food

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

The `ResourcefulSet` operator will now create a Pod for each `Food` in the system, and mount a volume on each one at `/etc/which-food`. Inside that directory, we'll have two files, named `name` and `type`, containing the data from the `Food` definition, much like it would if we had a `ConfigMap`.

If a `Food` item is created or deleted, the pods are killed and started appropriately; for updates, the pod is first killed, then a new one is started.
