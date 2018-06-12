import json

import flask
import peewee as pw

from model import APIKey, KVPair, create_key

app = flask.Flask(__name__)


@app.after_request
def add_no_coors(resp):
    '''
    adds the 'no cors' header to every response
    '''
    resp.headers['Access-Control-Allow-Origin'] = '*'
    return resp


@app.route('/api/newapikey/', methods=['POST'])
def handle_new_api_key():
    '''
    creates new api users
    '''
    data = dict(flask.request.json)
    user = APIKey()
    user.name = data.get('name', '')
    user.email = data.get('email', '')
    user.note = data.get('note', '')
    user.save()

    return json.dumps({
        'name': user.name,
        'email': user.email,
        'note': user.note,
        'apikey': user.key,
        'created': str(user.created)
    })


@app.route('/api/<string:api_key>/', methods=['GET'])
def handle_list_keys(api_key):
    '''
    get a list of keys for this user
    '''
    try:
        user = APIKey.get(APIKey.key == api_key)
    except APIKey.DoesNotExist:
        flask.abort(404)

    data = [{
        'key': p.key,
        'value': p.value,
        'created': str(p.created),
        'modified': str(p.modified)
    } for p in user.pairs]

    return json.dumps(data)


@app.route('/api/<string:api_key>/<string:key>/', methods=['GET'])
def handle_get_val(api_key, key):
    '''
    get a specific key-value pair for the user
    '''
    try:
        user = APIKey.get(APIKey.key == api_key)
    except APIKey.DoesNotExist:
        flask.abort(404)

    try:
        pair = KVPair.select().join(APIKey).where(APIKey.id == user.id,
                                                  KVPair.key == key).get()
    except KVPair.DoesNotExist:
        flask.abort(404)

    return json.dumps({
        'key': pair.key,
        'value': pair.value,
        'apikey': user.key,
        'created': str(pair.created),
        'modified': str(pair.modified)
    })


@app.route('/api/<string:api_key>/<string:key>/', methods=['PUT'])
def handle_put_val(api_key, key):
    '''
    create/overwrite a specific key-value pair for the user
    '''
    try:
        user = APIKey.get(APIKey.key == api_key)
    except APIKey.DoesNotExist:
        print(api_key)
        flask.abort(404)

    try:
        pair = KVPair.select().join(APIKey).where(APIKey.id == user.id,
                                                  KVPair.key == key).get()
    except KVPair.DoesNotExist:
        pair = KVPair()
        pair.owner = user
        pair.key = key
    finally:
        pair.value = flask.request.json['value']
        pair.save()

    return json.dumps({
        'key': pair.key,
        'value': pair.value,
        'apikey': user.key,
        'created': str(pair.created),
        'modified': str(pair.modified)
    })
