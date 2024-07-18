# How does it worK:

you can connect to it use `wscat`:

```shell
npx wscat -c ws://localhost:8080
```

upon connection you recvice your UUID:

```shell
< abc-xyz-123
```

This UUID is your `Group`. You can add other connections to your `Group` if you get their UUID. You can be added to someone else's group if they add your UUID to their `Group`

There are the following instructions available:

- `banner <info>`: This sets the value of `<info>` as the `banner` of your `Group`. Whenever you join the a `Group` you will recive their `banner`. Whenever others join your `Group` they will receive your `banner`.

- `join <uuid>`: With the join command you can join as a `peer` to the `Group` owned by the `<uuid>`. Joining the `<uuid>`'s `Group`, you will immediately receive their `banner`.

- `add <uuid>`: If you have someone else's `<uuid>`, you can add them to your `peers`. Adding them to your `peers` will send them your `banner`.

- `write <msg>`: You can write the `<msg>` to all the `peers` added to your `Group`.
