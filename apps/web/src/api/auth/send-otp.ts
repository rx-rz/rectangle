import { type UseMutationOptions, useMutation } from "@tanstack/react-query";

import { API_ROUTES } from "../api-routes";
import { type APIError, api } from "../index";

export type SendOTPInput = {
	email: string;
};

type Options = UseMutationOptions<string | undefined, APIError, SendOTPInput>;

export const sendOTPApi = async (data: SendOTPInput) => {
	return await api.message(API_ROUTES.auth.otp.send, "POST", {
		json: data,
	});
};

export const useSendOTPApi = (options?: Options) => {
	const mutation = useMutation({
		mutationFn: sendOTPApi,
		...options,
	});

	const { mutate, data, isPending, isSuccess } = mutation;

	return {
		...mutation,
		sendOTP: mutate,
		data,
		isPending,
		isSuccess,
	};
};
