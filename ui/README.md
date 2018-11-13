# Inbucket User Interface

This directory contains the source code for the Inbucket web user interface.
It is written in [Elm] 0.18, a *delightful language for reliable webapps.*

The UI was bootstrapped with [Create Elm App].

## Development

With `$INBUCKET` as the root of the git repository.

One time setup (assuming [Node.js] is already installed):

```
npm i create-elm-app@1.10.4 -g
```

In terminal 1 (inbucket daemon):

```
cd $INBUCKET/ui
elm-app build
cd $INBUCKET
make
etc/dev-start.sh
```

Inbucket will start, with HTTP listening on port 9000.  You may verify the web
UI is functional if this is your first time building Inbucket, but your dev/test
cycle should favor the development server below.

In terminal 2 (elm-app development server):

```
cd $INBUCKET/ui
elm-app start
```

[Create Elm App] will start a development HTTP server listening on port 3000.
You should use this server for UI development, as it features hot reload and the
Elm debugger.

[Create Elm App]: https://github.com/halfzebra/create-elm-app
[Elm]:            https://elm-lang.org
[Node.js]:        https://nodejs.org
