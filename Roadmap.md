# Vision

Fly.io has various blueprints, documentation pages, and blog pages telling you how to accomplish various tasks by writing code and using our APIs, headers, and the like. I want to explore augmenting this with working examples that include a small binary (currently ~11MB) in your Dockerfile along with a declarative description of your configuration. Optionally there may be startup scripts and shutdown hooks, but ultimately updating your configuration is as simple as:

```
fly deploy
```

# Overview

I built an application server for my [showcase](https://github.com/rubys/showcase?tab=readme-ov-file#showcase) application. Perhaps it may be of use to others. It still is a work in progress.

For purposes of this discussion, the application isn't important. Pick your favorite, smallish, web application. Perhaps a todo list. Backed by sqlite3.

Soon you find yourself in need for another todo list. There are a number of ways to implement this, but instead of making the database and/or the application more complicated, lets go with a separate database per todo list.

Java application servers like [Apache Tomcat](https://tomcat.apache.org/) solved this many years ago. You can mount multiple applications to different paths. So `/app1` could be one application, and `/app2` could be a second application. Paths can be anything, and even hierarchical, so `/sam/list1`, `/sam/list2`, `/sally/list1`, `/sally/list2` could be four todo lists. The applications could be the same, the difference being environment variables. By varying `DATABASE_URL`, the same application can be run, with each instance serving a different todo list.

The role of the application manager is to start web applications as needed, proxy requests to the correct application, stop individual applications when  they are no longer in use, and stop or suspend the entire application when all applications are not in use.

As your needs grow, it may be time to spread the applications across multiple machines. Microsoft Azure refers to this as a [Deployment Stamps pattern](https://learn.microsoft.com/en-us/azure/architecture/patterns/deployment-stamp). Perhaps put all of Sam's lists on one machine and all of Sally's lists on a second machine. These machines need not even be in close proximity, they can be placed close to their owners. No application changes are required, Sam's machine can be configured to serve Sam's lists, and replay requests for Sally to her machine, and vice versa.

Next let's look at startup. Each machine may need to generate its own configuration file, there may be database migrations, and there may be other activities that you want to perform before your server becomes live. I handle this by starting the server with a [maintenance configuration](https://github.com/rubys/showcase/blob/main/config/navigator-maintenance.yml) that responds to all requests with a [html page](https://github.com/rubys/showcase/blob/main/public/503.html) saying that updates are being installed and configured to refresh the page using [html meta tags](https://developer.mozilla.org/en-US/docs/Web/HTML/Reference/Elements/meta/http-equiv#refresh). Once the machine is ready, I reload the updated configuration and it is off to the races.

Additionally, at startup you may want to start other processes (things like redis, workers, etc). I mentioned stopping individual applications and stopping or suspending application servers themselves - there may be processes you want to spawn at these times too (e.g. backup).

Authentication support is built in - Navigator will authenticate users, once authenticated, web applications will be started and those applications will be responsible for access control.

Finally, there are all of the things you would expect or need from a web server, things like static file support, websocket upgrade, and rewrite rules.

## Historical background

While the application server itself is new, I've been running with this [architecture](https://github.com/rubys/showcase/blob/main/ARCHITECTURE.md) for years, using [nginx](https://nginx.org/) and [Phusion Passenger](https://www.phusionpassenger.com/). I have my configuration in a database and at build time, produce a yaml file from it. At startup, each machine parses that yaml file and produces a nginx conf file.

 I currently have 75 different dance studios across 8 countries and 4 continents using this software, with 339 individual events where each event is a database. I've been organizing studios by fly.io regions.

The primary motivation for the creation of the Navigator tool is twofold: reduce complexity by producing a configuration file that can be directly loaded, and to avoid limitations of the nginx server.

As an example: I want to make replay smarter: the normal case of a misdirected request is that a new user makes a request and fly.io's proxy simply picks a random machine in a nearby region. In such case, a replay is all that is needed. But there also are cases like fly deploy where the destination isn't ready yet, and this can be determined by dynamically checking [internal DNS](https://fly.io/docs/networking/private-networking/#fly-io-internal-dns). Eventually that could lead to starting new machines dynamically, but for now the plan is to have one machine per user, where that machine is stopped or suspended when not in use.

## Current status

I'm still running nginx and Passenger in production, but I have a second application using Navigator that is nearly fully functional. The one known feature that I will need to complete before putting this into production myself is hooks that are called when individual applications stop or entire application servers suspend.

I'm looking to change my configuration from one machine per region to one machine per user, taking advantage of the recent support for [routing to preferred instances](https://community.fly.io/t/routing-to-preferred-instances/25686).

Up to now, I've been focusing on a configuration file that can be generated programmatically and match my current implementation. The result still has a distinctive nginx flavor with concepts like try_files and regular expressions for paths. I plan to review this and work to make a format that can be manually edited. There undoubtedly are a number of Rails specific assumptions in the current implementation, the plan would be to identify them and move such to the configuration.

I'm currently [building the Navigator](https://github.com/rubys/showcase/blob/ab75c95765554babaf1b6a4d1f97440e8491b63e/Dockerfile.nav#L51-L62) as a part of the deployment of my Showcase application, but once it is ready, I plan to make releases to Dockerhub so that including it in your images will be as easy as:

```dockerfile
COPY --from=rubys/navigator /navigator /usr/local/bin/navigator
```

