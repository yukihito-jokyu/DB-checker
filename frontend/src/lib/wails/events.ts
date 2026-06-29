import { EventsOn } from "@wails/runtime/runtime";

export type WailsEventHandler<TPayload = unknown> = (payload: TPayload) => void;

export type WailsEventUnsubscribe = () => void;

/** Wails Events の購読を開始し、解除関数を返す。 */
export function subscribeWailsEvent<TPayload = unknown>(
	eventName: string,
	handler: WailsEventHandler<TPayload>,
): WailsEventUnsubscribe {
	return EventsOn(eventName, (payload: TPayload) => {
		handler(payload);
	});
}
