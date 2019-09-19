# kvss
Simple and easy to use key-value store.

### api

    POST /api/newapikey/
        expects json
        { 
            "name": <string>,
            "email": <string>,
            "note": <string>
        }

        returns json
        {
            "name": <string>,
            "email": <string>,
            "note": <string>,
            "apikey": <string>,
            "created": <datetime>
        }

    GET /api/<string:apikey>/
        returns json
        [
            {
                "key": <string>,
                "value": <string>,
                "created": <datetime>,
                "modified": <datetime>
            }
        ]

    GET /api/<string:apikey>/<string:key>/
        returns json
        {
            "key": <string>,
            "value": <string>,
            "apikey": <string>,
            "created": <string>,
            "modified": <string>
        }

    PUT /api/<string:apikey>/<string:key>/
        expects json. max size of string 4096kb
        { "value": <string> }

        returns json
        {
            "key": <string>,
            "value": <string>,
            "apikey": <string>,
            "created": <datetime>,
            "modified": <datetime>
        }

    