# tcp-chat
Common Chat Room

TCP Chat
========

Should be plaintext and relatively usable with netcat, telnet or similar utilities when used with terminal emulator having UNIX newlines (\n)

No authentication or authorization is implied. 

Server
------
- Listens on port 20000/tcp
- Accepts connections from clients
- Relays message of each client to other clients
- Notifies all clients when other client joins or leaves
- Maintains timeouts for inactive users, disconnecting them
- Maintains history of messages and join/part events in memory, supplying new client with history of past 10 messages

Protocol
======== 

Protocol data
=============

Partial message
---------------
- Any UTF-8 encoded payload of TCP packet sent by client

Complete message
----------------
- A UTF-8 encoded string in format "`[$TIME] $AUTHOR $BODY\n`" sent by server
- `$TIME` is time of day in format "`$0_padded_24_hour:$0_padded_minute:$0_padded_second`" at moment of receiving first non-empty partial message or event generation   
- `$AUTHOR` is in format "`<$IDENTITY>`" of message belongs to user with identity `$IDENTITY`, or "`**SERVER**`" if message sent due to server-generated event.
- `$BODY` is the body of complete message
- \n being a UNIX newline

Protocol events
===============

PARTIAL_MESSAGE event
---------------------
- Happens whenever client sends data over TCP connection, packet payload being message content
- Raw partial message could not be longer than 2048 bytes (likely less than 1500 due to IP fragmentation).
- Complete control characters are filtered out of message content.
- Empty (after filtering) messages are not kept, but do prevent client from timing out just as any other.
- Non-empty partial messages are buffered
- Up to 8 messages are buffered.

COMPLETE_MESSAGE event
----------------------
- Happens whenever last non-empty message in partial message buffer gets older than 100ms since receiving it.
- Happens whenever partial message buffer is filled full with messages.  
- All partial messages in buffer are concatenated with following rules:
  * Starting with empty `$complete` string buffer and empty `$reminder` byte buffer
  * Current partial message being referenced ad `$i_partial`
  * Copy all valid UTF-8 sequences from concatenation of (`$reminder` + `$i_partial`), while skipping all invalid UTF-8 sequences and control characters, to `$complete` buffer
  * If `$i_partial` ends with invalid UTF-8 sequence, put that last invalid UTF-8 sequence in `$reminder` buffer, otherwise empty `$reminder` buffer  
- `$complete` should be filtered out of any control characters that was split into different partial messages.
- Complete message with body `$complete`, identity of sending client and timestamp of first partial message then gets relayed to all clients
- Server could generate complete messages on it's own on other events.

JOIN event
----------
- Happens whenever client establishes TCP connnection.
- Client `IDENTITY` = sha1(`$IP` + ':' + `$PORT`).
- `$IP` is IP-address of connected user
- `$PORT` is TCP port of connected user 
- Just joined client receives 10 last messages from server history
- All clients (including just joined one) receive complete message with body "Client `$IDENTITY` has joined" 

PART event
----------
- Happens whenever client closes TCP connection or fails to maintain activity within timeout of 60 seconds
- All remaining clients receive complete message with body "Client `$IDENTITY` has `$ACTION`"
- If user gracefully terminated TCP connection, `ACTION` = "left"
- If user were disconnected due to time out, `ACTION` = "timed out"

KEEPALIVE event
--------------
- Happens whenever cleint sends TCP packet to server, regardless of it's payload
- Prevents client from timing out
