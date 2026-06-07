import { type UseMutationOptions, useMutation } from "@tanstack/react-query";

import { API_ROUTES } from "../api-routes";
import { type APIError, api } from "../index";
import { queries } from "../queries";
import type { SignupWithEmailResponse } from "./signup-with-email";

export type LoginWithEmailInput = {
	email: string;
	password: string;
};

type Options = UseMutationOptions<
	SignupWithEmailResponse,
	APIError,
	LoginWithEmailInput
>;

export const loginWithEmailApi = async (data: LoginWithEmailInput) => {
	return await api.request<SignupWithEmailResponse>(
		API_ROUTES.auth.login,
		"POST",
		{
			json: data,
		},
	);
};

export const useLoginWithEmailApi = (options?: Options) => {
	const mutation = useMutation({
		mutationKey: queries.auth.loginWithEmail.queryKey,
		mutationFn: loginWithEmailApi,
		...options,
	});

	const { mutate, data, isPending, isSuccess } = mutation;

	return {
		...mutation,
		loginWithEmail: mutate,
		data,
		isPending,
		isSuccess,
	};
};
