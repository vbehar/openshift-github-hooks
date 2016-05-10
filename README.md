# OpenShift GitHub Hooks

**Manages GitHub hooks for OpenShift BuildConfig triggers.**

[![DockerHub](https://img.shields.io/badge/docker-vbehar%2Fopenshift--github--hooks-008bb8.svg)](https://hub.docker.com/r/vbehar/openshift-github-hooks/)
[![Travis](https://travis-ci.org/vbehar/openshift-github-hooks.svg?branch=master)](https://travis-ci.org/vbehar/openshift-github-hooks)
[![Circle CI](https://circleci.com/gh/vbehar/openshift-github-hooks/tree/master.svg?style=svg)](https://circleci.com/gh/vbehar/openshift-github-hooks/tree/master)

This [Go](http://golang.org/) application is an [OpenShift](http://www.openshift.org/) client that can manage your [GitHub Webhooks](https://developer.github.com/v3/repos/hooks/) for your OpenShift [BuildConfig](https://docs.openshift.org/latest/dev_guide/builds.html#defining-a-buildconfig) triggers.

Its main feature is to keep your GitHub Webhooks in sync with your OpenShift BuildConfigs, and so automatically create/delete the webhooks on GitHub to reflect the [build trigger](https://docs.openshift.org/latest/dev_guide/builds.html#webhook-triggers) changes on OpenShift.

It is very useful when you host your source code on a GitHub organization: instead of having to manually create your webhooks on GitHub when you create a new build on OpenShift (and then forget to delete it when you remove the build on OpenShift), you just need to run this application (on OpenShift), and let it handle all that boring stuff for you!

There are also additional commands, for example to list all the Hooks of a GitHub organization that references your OpenShift instance.

## How It Works

### Keep Webhooks in sync

This application can be deployed on OpenShift or anywhere else, but I guess you will want to run it in your OpenShift cluster - at least for the `sync` command that runs as a daemon. It should run with a [ServiceAccount](https://docs.openshift.org/latest/architecture/core_concepts/projects_and_users.html#users) that has the `cluster-reader` [role](https://docs.openshift.org/latest/architecture/additional_concepts/authorization.html#roles), so that it can watch all the [BuildConfigs](https://docs.openshift.org/latest/dev_guide/builds.html#defining-a-buildconfig).

The `sync` command will listen for every BuildConfig change in the cluster, and for all BuildConfig with a [GitHub Webhook trigger](https://docs.openshift.org/latest/dev_guide/builds.html#webhook-triggers), it will try to [create the hook on the GitHub repository](https://developer.github.com/v3/repos/hooks/#create-a-hook), using the [GitHub API](https://developer.github.com/v3/).

It will also list all the existing webhooks on GitHub, and remove webhooks that references non-existing OpenShift BuildConfigs.

At a pre-defined period interval, it will re-sync everything, to make sure it didn't miss any event.

#### Exceptions

If you want to bypass this automatic hook creation for a specific BuildConfig, you can just set the `openshift-github-hooks-sync/ignore` annotation to `true`:

```
kind: BuildConfig
apiVersion: v1
metadata:
  annotations:
  	openshift-github-hooks-sync/ignore: "true"
[...]
```

With this annotation (and its value set to `true`), no GitHub Webhook will be created/deleted.

### Listing Webhooks

The `list` command will just use the GitHub API to list webhooks and print them in the standard output.

It uses a [GitHub Access Token](https://help.github.com/articles/creating-an-access-token-for-command-line-use/) to talk to the GitHub API. You can create such a token in your [GitHub Tokens Settings](https://github.com/settings/tokens) page. It requires the `repo` and `admin:repo_hook` scopes, to be able to list repositories, and list/create/delete hooks.

## Usage

This application has many commands, you can list them by running:

```
openshift-github-hooks -h
```

### Running on OpenShift

If you want to deploy this application on an OpenShift cluster to run the `sync` command as a controller, you need to:

* create a project named `github-hooks-controller`:

  ```
  oc new-project github-hooks-controller
  ```

* create a specific [ServiceAccount](https://docs.openshift.org/latest/architecture/core_concepts/projects_and_users.html#users) named `github-hooks-controller`, using the provided [openshift-serviceaccount.yml](openshift-serviceaccount.yml) definition file:

  ```
  oc create -f openshift-serviceaccount.yml
  ```

* as a cluster admin, give the `cluster-reader` role to your new ServiceAccount:

  ```
  oadm policy add-cluster-role-to-user cluster-reader system:serviceaccount:github-hooks-controller:github-hooks-controller
  ```

* create a new application from the provided [openshift-template-deploy-only.yml](openshift-template-deploy-only.yml) template, and overwrite some parameters:

  ```
  oc new-app -f openshift-template-deploy-only.yml -p SERVICE_ACCOUNT=github-hooks-controller,GITHUB_ACCESS_TOKEN=xxx,GITHUB_ORGANIZATION=yyy
  ```

Of course, replace `xxx` by the value of your [GitHub Access Token](https://help.github.com/articles/creating-an-access-token-for-command-line-use/). To create such a token, go to your [GitHub Tokens Settings](https://github.com/settings/tokens) page, and create a new token with the `repo` and `admin:repo_hook` scopes.
You also need to define the GitHub organization name for which the controller will manage the hooks.

You can use either of the following templates:

* [openshift-template-deploy-only.yml](openshift-template-deploy-only.yml) to just deploy from an existing Docker image - by default [vbehar/openshift-github-hooks](https://hub.docker.com/r/vbehar/openshift-github-hooks/)
* [openshift-template-full.yml](openshift-template-full.yml) to build from sources (by default the [vbehar/openshift-github-hooks](https://github.com/vbehar/openshift-github-hooks) github repository) and then deploy

### Running locally

If you want to run it on your laptop:

* Install [Go](http://golang.org/) (tested with 1.6) and [setup your GOPATH](https://golang.org/doc/code.html)
* clone the sources in your `GOPATH`

  ```
  git clone https://github.com/vbehar/openshift-github-hooks.git $GOPATH/src/github.com/vbehar/openshift-github-hooks
  ```

* install [godep](https://github.com/tools/godep) (to use the vendored dependencies)

  ```
  go get github.com/tools/godep
  ```

* build the binary with godep:

  ```
  cd $GOPATH/src/github.com/vbehar/openshift-github-hooks
  godep go install
  ```

* have a look at the application commands, help messages and examples:

  ```
  $GOPATH/bin/openshift-github-hooks -h
  $GOPATH/bin/openshift-github-hooks list -h
  $GOPATH/bin/openshift-github-hooks sync -h
  ```

  * if you want to run the `sync` command, you will need to get your [GitHub Access Token](https://help.github.com/articles/creating-an-access-token-for-command-line-use/) and the GitHub organization name, and then run:

    ```
    $GOPATH/bin/openshift-github-hooks sync --github-token="..." --organization="..."
    ```

  * for the `list` command, you will also need to get your [GitHub Access Token](https://help.github.com/articles/creating-an-access-token-for-command-line-use/) and the GitHub organization name, and then run:

    ```
    $GOPATH/bin/openshift-github-hooks list --github-token="..." --organization="..."
    ```

* enjoy!

## License

Copyright 2016 the original author or authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.