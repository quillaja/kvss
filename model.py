import crypt
import datetime as dt
import random

import peewee as pw

_DB = pw.SqliteDatabase('kvss.db')


def create_key(length=32):
    alphabet = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
    key = [alphabet[random.randrange(0, len(alphabet))] for i in range(length)]
    return ''.join(key)


class Base(pw.Model):
    '''
    Base model.
    '''
    created = pw.DateTimeField(default=dt.datetime.now)
    modified = pw.DateTimeField()

    def save(self, *args, **kwargs):
        self.modified = dt.datetime.now()
        super().save(*args, **kwargs)

    class Meta(object):
        database = _DB


class APIKey(Base):
    '''
    Represents users by their unique API key.
    '''
    name = pw.CharField(null=True)
    email = pw.CharField(null=True)
    key = pw.CharField(unique=True, default=create_key)
    note = pw.TextField(null=True)


class KVPair(Base):
    '''
    Represents the data, a string-string key-value pair. The 'owner' field 
    should point back to the APIKey that owns this kv-pair.
    '''
    owner = pw.ForeignKeyField(rel_model=APIKey, related_name='pairs')
    key = pw.CharField()
    value = pw.TextField()


def create_tables():
    _DB.connect()
    _DB.create_tables([APIKey, KVPair], safe=True)
    _DB.close()
