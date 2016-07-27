#!/usr/bin/env bash

cd mauditnozzle

cf api api.walnut.cf-app.com
cf login -u admin -p $PASSWORD
cf