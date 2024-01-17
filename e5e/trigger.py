#!/usr/bin/env python3

import os
import base64
import json
import urllib.request

def trigger(event, context):
    if not _check_auth(event):
        return {
            "status": 401,
            "data":  "Authentication required",
            "type":  "text",
        }

    _trigger_goanxio_workflow()

    return {
        "status": 200,
        "data":  "OK",
        "type":  "text",
    }

def _check_auth(event):
    cfg_token = os.getenv("GOANXIO_E5E_TOKEN")

    if    not event                      \
       or not 'request_headers' in event \
       or not 'authorization'   in event['request_headers']:
        return False

    auth       = event['request_headers']['authorization']
    auth_parts = auth.split(" ", 2)

    return len(auth_parts) == 2             \
           and auth_parts[0] == "Bearer"    \
           and auth_parts[1] == cfg_token

def _trigger_goanxio_workflow():
    cfg_gh_token_user = "anx-release"
    cfg_gh_token      = os.getenv("GOANXIO_TOKEN")

    # using HTTPBasicAuthHandler does not seem to work with github, probably due to
    # no X-WWW-Authenticate header being sent by it.
    auth         = ('%s:%s' % (cfg_gh_token_user, cfg_gh_token))
    encoded_auth = base64.b64encode(auth.encode('ascii')).decode('ascii')

    url = 'https://api.github.com/repos/anexia/go.anx.io/dispatches'

    request = urllib.request.Request(
        url,
        headers={
            'Content-Type': 'application/json; charset=utf-8',
            'Authorization': 'Basic ' + encoded_auth,
        },
        data=json.dumps({
            "event_type": "rebuild_pages",
        }).encode('utf-8'),
    )

    response = urllib.request.urlopen(request)
