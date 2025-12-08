import { Injectable } from '@nestjs/common';
import { Server, WebSocket } from 'ws';

interface AuthenticatedWebSocket extends WebSocket {
  userId?: string;
}

@Injectable()
export class NotificationGateway {
  private server: Server;
  private clients: Map<string, Set<AuthenticatedWebSocket>> = new Map();

  initialize(server: Server) {
    this.server = server;

    this.server.on('connection', (client: AuthenticatedWebSocket, request: any) => {
      this.handleConnection(client, request);
    });
  }

  handleConnection(client: AuthenticatedWebSocket, request: any) {
    const url = new URL(request.url, `http://${request.headers.host}`);
    const userId = url.searchParams.get('user_id');

    if (userId) {
      client.userId = userId;

      if (!this.clients.has(userId)) {
        this.clients.set(userId, new Set());
      }
      this.clients.get(userId).add(client);

      console.log(`Client connected: userId=${userId}, total=${this.clients.get(userId).size}`);

      client.on('close', () => this.handleDisconnect(client));
    } else {
      client.close(4000, 'Missing userId');
    }
  }

  handleDisconnect(client: AuthenticatedWebSocket) {
    if (client.userId) {
      const userClients = this.clients.get(client.userId);
      if (userClients) {
        userClients.delete(client);
        if (userClients.size === 0) {
          this.clients.delete(client.userId);
        }
      }
      console.log(`Client disconnected: userId=${client.userId}`);
    }
  }

  sendToUser(userId: string, data: any) {
    const userClients = this.clients.get(userId);
    if (userClients && userClients.size > 0) {
      const message = JSON.stringify(data);
      userClients.forEach(client => {
        if (client.readyState === WebSocket.OPEN) {
          client.send(message);
        }
      });
      return true;
    }
    return false;
  }

  sendToUsers(userIds: string[], data: any) {
    const message = JSON.stringify(data);
    let sentCount = 0;

    userIds.forEach(userId => {
      const userClients = this.clients.get(userId);
      if (userClients) {
        userClients.forEach(client => {
          if (client.readyState === WebSocket.OPEN) {
            client.send(message);
            sentCount++;
          }
        });
      }
    });

    return sentCount;
  }

  isUserOnline(userId: string): boolean {
    return this.clients.has(userId) && this.clients.get(userId).size > 0;
  }
}
