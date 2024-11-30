#ifdef _WIN32
#include <winsock2.h>
#include <ws2tcpip.h>
#include <pthread.h>
#else
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include <arpa/inet.h>
#include <pthread.h>
#endif

#define SERVER_PORT 8080
#define RECEIVE_BUFFER_SIZE 1024

// Function to handle client requests
void *handle_client_request(void *client_socket_ptr) {
    int client_socket = *((int *)client_socket_ptr);
    free(client_socket_ptr);  // Free the allocated memory for the client socket

    char request_buffer[RECEIVE_BUFFER_SIZE] = {0};
    const char *http_response =
        "HTTP/1.1 200 OK\r\n"
        "Content-Type: text/html\r\n"
        "Connection: close\r\n\r\n"
        "<html><body><h1>Hello, World!</h1></body></html>";

    // Read the client's HTTP request
    read(client_socket, request_buffer, RECEIVE_BUFFER_SIZE);
    printf("Received HTTP request:\n%s\n", request_buffer);

    // Send the HTTP response back to the client
    write(client_socket, http_response, strlen(http_response));

    // Close the client connection
#ifdef _WIN32
    closesocket(client_socket);  // Use closesocket on Windows
#else
    close(client_socket);  // Use close on Unix-like systems
#endif

    return NULL;
}

int main() {
    int server_socket_file_descriptor, *client_socket_file_descriptor;
    struct sockaddr_in server_address, client_address;
    socklen_t client_address_length = sizeof(client_address);
#ifdef _WIN32
    WSADATA wsaData;
    pthread_t thread_id;

    // Initialize Winsock (for Windows)
    if (WSAStartup(MAKEWORD(2, 2), &wsaData) != 0) {
        printf("Winsock initialization failed\n");
        return 1;
    }
#else
    pthread_t thread_id;
#endif

    // Create the server socket
    if ((server_socket_file_descriptor = socket(AF_INET, SOCK_STREAM, 0)) == 0) {
        perror("Failed to create server socket");
        exit(EXIT_FAILURE);
    }

    // Configure the server address
    server_address.sin_family = AF_INET;
    server_address.sin_addr.s_addr = INADDR_ANY;
    server_address.sin_port = htons(SERVER_PORT);

    // Bind the server socket to the specified address and port
    if (bind(server_socket_file_descriptor, (struct sockaddr *)&server_address, sizeof(server_address)) < 0) {
        perror("Failed to bind server socket to address");
#ifdef _WIN32
        WSACleanup();  // Clean up Winsock
#endif
        close(server_socket_file_descriptor);
        exit(EXIT_FAILURE);
    }

    // Start listening for incoming connections
    if (listen(server_socket_file_descriptor, 3) < 0) {
        perror("Failed to start listening on server socket");
#ifdef _WIN32
        WSACleanup();  // Clean up Winsock
#endif
        close(server_socket_file_descriptor);
        exit(EXIT_FAILURE);
    }

    printf("HTTP server is running on port %d\n", SERVER_PORT);

    // Infinite loop to accept and handle incoming connections
    while (1) {
        // Accept an incoming client connection
        client_socket_file_descriptor = malloc(sizeof(int));  // Allocate memory for the client socket
        if ((*(client_socket_file_descriptor) = accept(server_socket_file_descriptor, (struct sockaddr *)&client_address, &client_address_length)) < 0) {
            perror("Failed to accept client connection");
            free(client_socket_file_descriptor);  // Free memory on error
            continue;
        }

        // Create a new thread to handle the client's request
        if (pthread_create(&thread_id, NULL, handle_client_request, client_socket_file_descriptor) != 0) {
            perror("Failed to create thread");
#ifdef _WIN32
            closesocket(*(client_socket_file_descriptor));  // Close the socket if thread creation fails
#else
            close(*(client_socket_file_descriptor));  // Use close on Unix-like systems
#endif
            free(client_socket_file_descriptor);     // Free the allocated memory
            continue;
        }

        // Detach the thread so it can clean up its resources automatically when done
        pthread_detach(thread_id);
    }

    // Close the server socket (this will never be reached in this example)
    close(server_socket_file_descriptor);
#ifdef _WIN32
    WSACleanup();  // Clean up Winsock
#endif
    return 0;
}
