import { type UseMutationOptions, useMutation } from "@tanstack/react-query";

import { API_ROUTES } from "../api-routes";
import { type APIError, api } from "../index";
import { queries } from "../queries";

export type SignupWithEmailInput = {
	name?: string | null;
	email: string;
	password: string;
};

export type AuthUser = {
	id: string;
	name?: string | null;
	email: string;
	avatar_url?: string | null;
	email_verified_at?: string | null;
	created_at: string;
};

export type AuthSession = {
	id: string;
	expires_at: string;
};

export type SignupWithEmailResponse = {
	user: AuthUser;
	session?: AuthSession | null;
};

type Options = UseMutationOptions<
	SignupWithEmailResponse,
	APIError,
	SignupWithEmailInput
>;

export const signupWithEmailApi = async (data: SignupWithEmailInput) => {
	return await api.request<SignupWithEmailResponse>(
		API_ROUTES.auth.signup,
		"POST",
		{
			json: data,
		},
	);
};

export const useSignupWithEmailApi = (options?: Options) => {
	const mutation = useMutation({
		mutationKey: queries.auth.signupWithEmail.queryKey,
		mutationFn: signupWithEmailApi,
		...options,
	});

	const { mutate, data, isPending, isSuccess } = mutation;

	return {
		...mutation,
		signupWithEmail: mutate,
		data,
		isPending,
		isSuccess,
	};
};
