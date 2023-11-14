

Usage instructions:
1. Open three different terminals.
2. In each terminal navigate to the DisMutExc folder.
3. In each terminal type the following command to run the code.

        go run Peer.go
4. Once the code runs you will be prompted to "Please enter the Node ID". It is IMPORTANT that you have all three terminals ready with this prompt before moving on
5. You can now enter the id '0' into the first terminal, '1' into the second terminal, and '2' into the third terminal. This needs to happen within 10 seconds. Otherwise, the system will crash due to being unable to connect to the other ports.

Each peer will now send, receive, and handle critical section requests.

The system will only run through a single cycle of sending requests, but will handle all incoming requests by other peers.

6. When done, you can freely terminate each peer and observe the generated log- and data file. 

The data file should contain a single "Hello World" from each peer.



