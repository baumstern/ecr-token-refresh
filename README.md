ecr-token-refresh
---

Refresh ECR token as Kubernetes Secret used to[`imagePullSecrets`].
It creates a secret of [`kubernetes.io/dockerconfigjson`] type.



### Environment variables

Name                    | Required | Description                                          | Default
------------------------|----------|------------------------------------------------------|-------
`AWS_REGION`            |    yes   | AWS region of ECR registry                           | -
`AWS_ACCESS_KEY_ID`     |    yes   | AWS access key associated with an IAM user or role   | -
`AWS_SECRET_ACCESS_KEY` |    yes   | the secret key associated with the access key        | -
`KUBE_SECRET_NAME`      |    no    | Name of the Secret contains image pull credential    | ecr-pull-secret-$AWS_REGION
`KUBE_NAMESPACE`        |    no    | Namespace which secret applied to                    | default


### Required IAM permission for `AWS_ACCESS_KEY_ID`

* [`AmazonEC2ContainerRegistryReadOnly`]


## Usage

Below shows creating ECR token in default namespace.

Create a secret of IAM credential:
```bash
kubectl create secret \
          generic ecr-credential \
          --from-literal=REGION=<YOUR_AWS_REGION> \
          --from-literal=AWS_ACCESS_KEY_ID=<YOUR_AWS_ACCESS_KEY_ID> \
          --from-literal=AWS_SECRET_ACCESS_KEY=<YOUR_AWS_SECRET_ACCESS_KEY> \
          --from-literal=KUBE_NAMESPACE=default \
          --namespace=default
``` 

Create a Service Account to authorize CronJob:
```bash
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: ServiceAccount
metadata:
  name: svac-ecr
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: role-ecr
rules:
  - apiGroups: [""]
    resources:
      - secrets
    verbs:
      - get
      - create
      - delete
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: rb-ecr
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: role-ecr
subjects:
  - kind: ServiceAccount
    name: svac-ecr
    
---
EOF
```

Create ECR token refresh CronJob which runs every 6 hours:
```bash
cat <<EOF | kubectl apply -f -
apiVersion: batch/v1beta1
kind: CronJob
metadata:
  name: cronjob-ecr-token-refresh
spec:
  schedule: "* */6 * * *"
  successfulJobsHistoryLimit: 3
  failedJobsHistoryLimit: 5
  jobTemplate:
    spec:
      template:
        spec:
          restartPolicy: Never
          serviceAccountName: svac-ecr
          containers:
            - name: ecr-token-refresh
              image: ghcr.io/gurrpi/ecr-token-refresh:v0.1.1
              imagePullPolicy: IfNotPresent
              env:
                - name: AWS_REGION
                  valueFrom:
                    secretKeyRef:
                      key: REGION
                      name: ecr-credential
                - name: AWS_ACCESS_KEY_ID
                  valueFrom:
                    secretKeyRef:
                      key: AWS_ACCESS_KEY_ID
                      name: ecr-credential
                - name: AWS_SECRET_ACCESS_KEY
                  valueFrom:
                    secretKeyRef:
                      key: AWS_SECRET_ACCESS_KEY
                      name: ecr-credential
                - name: KUBE_NAMESPACE
                  valueFrom:
                    secretKeyRef:
                      key: KUBE_NAMESPACE
                      name: ecr-credential
      backoffLimit: 1
EOF
```

## Compatibility

Developed for Kubernetes version `v1.18`. Other minor version may not work.

 
## Alternatives

* https://github.com/anaganisk/ecr-kube-helper
* https://github.com/skuid/ecr-token-refresh


[`imagePullSecrets`]: https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/#create-a-pod-that-uses-your-secret
[`kubernetes.io/dockerconfigjson`]: https://kubernetes.io/docs/concepts/configuration/secret/#docker-config-secrets
[`AmazonEC2ContainerRegistryReadOnly`]: https://docs.aws.amazon.com/AmazonECR/latest/userguide/ecr_managed_policies.html#AmazonEC2ContainerRegistryReadOnly