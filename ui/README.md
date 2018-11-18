# Inbucket User Interface

This directory contains the source code for the Inbucket web user interface.
It is written in [Elm] 0.19, a *delightful language for reliable webapps.*

## Development

With `$INBUCKET` as the root of the git repository.

One time setup (assuming [Node.js] is already installed):

```
cd $INBUCKET/ui
npm i elm -g
npm i
npm run build
```

This will the create `node_modules`, `elm-stuff`, and `dist` directories.

### Terminal 1: inbucket daemon

```
cd $INBUCKET
make
etc/dev-start.sh
```

Inbucket will start, with HTTP listening on port 9000.  You may verify the web
UI is functional if this is your first time building Inbucket, but your dev/test
cycle should favor the development server below.

### Terminal 2: webpack development server

```
cd $INBUCKET/ui
npm run dev
```

npm will start a development HTTP server listening on port 3000.  You should use
this server for UI development, as it features hot reload and the Elm debugger.

[Elm]:            https://elm-lang.org
[Node.js]:        https://nodejs.org
