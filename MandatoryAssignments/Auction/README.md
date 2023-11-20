Usage instructions:
1. Open a new terminal and navigate to the appropriate folder, "Auction".
2. In the same terminal, use the following command to start the server. (Note that the server must be started before any client)

        go run Server.go
Repeat step 1 and 2 for each new server you want to start.
For each new redundant server, a crash can be handled. Any amount of servers can be started,
but it is recommended to start less than 10 servers to avoid any failures during discovery.
If a crash occurs a new server can be started and will automatically join the network.



3. In a separate terminal, navigate to the same folder.
4. Use the following command in the new folder to join the server with a new client.

        go run Client.go
5. Once the prompt to enter your name appears, enter your preferred name.

If you get the message "Connected to server on x" then you are ready to proceed to start placing bids

Repeat step 3, 4 and 5 for each new client you want to start.

Once the item has been sold, the clients will automatically terminate.