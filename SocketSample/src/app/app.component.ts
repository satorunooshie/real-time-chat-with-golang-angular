import { Component, OnInit, OnDestroy } from '@angular/core';
import { SocketService } from "./socket.service";

@Component({
  selector: 'app-root',
  templateUrl: './app.component.html',
  styleUrls: ['./app.component.css']
})
export class AppComponent {
  title = 'SocketSample';

  public messages: Array<any>;
  public chatBox: string;

  public constructor(private socket: SocketService) {
    this.messages = [];
    this.chatBox = "";
  }

  public ngOnInit() {
    this.socket.getEventListener().subscribe(event => {
      switch (event.type) {
        case "message":
          let data = event.data.content;
          if (event.data.sender) {
            data = event.data.sender + ": " + data;
          }
          this.messages.push(data);
          break;
        case "close":
          this.messages.push("/The socket connection has been closed");
          break;
        case "open":
          this.messages.push("/The socket connection has been established");
          break;
      }
    });
  }

  public ngOnDestroy() {
    this.socket.close();
  }

  public send() {
    if (this.chatBox) {
      this.socket.send(this.chatBox);
      this.chatBox = "";
    }
  }

  public isSystemMessage(message: string) {
    return message.startsWith("/") ? "<strong>" + message.substring(1) + "</strong>" : message;
  }
}
