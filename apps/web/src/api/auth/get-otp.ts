import { type UseMutationOptions, useMutation } from "@tanstack/react-query";

import { API_ROUTES } from "../api-routes";
import { type APIError, api } from "../index";
import { queries } from "../queries";

export type GetOTPInput = {
	email: string;
};

type Options = UseMutationOptions<string | undefined, APIError, GetOTPInput>;

export const getOTPApi = async (data: GetOTPInput) => {
	return await api.message(API_ROUTES.auth.otp.send, "POST", {
		json: data,
	});
};

export const useGetOTPApi = (options?: Options) => {
	const mutation = useMutation({
		mutationKey: queries.auth.getOTP.queryKey,
		mutationFn: getOTPApi,
		...options,
	});

	const { mutate, data, isPending, isSuccess } = mutation;

	return {
		...mutation,
		getOTP: mutate,
		data,
		isPending,
		isSuccess,
	};
};
