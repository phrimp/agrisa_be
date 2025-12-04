export interface ISubscriberService {
  registerSubscriber(data: any): Promise<any>;
  updateSubscriber(data: any): Promise<any>;
  getSubscribersByUserId(userId: string): Promise<any[]>;
  getAllSubscribers(): Promise<any[]>;
  unsubscribe(userId: string, type: string): Promise<void>;
  deleteSubscriber(id: string): Promise<void>;
}
