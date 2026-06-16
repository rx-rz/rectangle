import { type UseMutationOptions, useMutation } from "@tanstack/react-query";

import { API_ROUTES } from "../api-routes";
import { type APIError, api } from "../index";
import { queries } from "../queries";
import type { SignupWithEmailResponse } from "./signup-with-email";

export type VerifyOTPInput = {
	email: string;
	code: string;
};

type Options = UseMutationOptions<
	SignupWithEmailResponse,
	APIError,
	VerifyOTPInput
>;

export const verifyOTPApi = async (data: VerifyOTPInput) => {
	return await api.request<SignupWithEmailResponse>(
		API_ROUTES.auth.otp.verify,
		"POST",
		{
			json: data,
		},
	);
};

export const useVerifyOTPApi = (options?: Options) => {
	const mutation = useMutation({
		mutationKey: queries.auth.verifyOTP.queryKey,
		mutationFn: verifyOTPApi,
		...options,
	});

	const { mutate, data, isPending, isSuccess } = mutation;

	return {
		...mutation,
		verifyOTP: mutate,
		data,
		isPending,
		isSuccess,
	};
};
