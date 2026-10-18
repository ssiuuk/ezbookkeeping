import logger from '@/lib/logger.ts';
import type { ApiResponsePromise } from '@/lib/services.ts';

// unwraps an api response, rejecting with a message or the server error the ui can show
export function handleApiResponse<T>(promise: ApiResponsePromise<T>, logMessage: string, defaultErrorMessage: string): Promise<T> {
    return new Promise((resolve, reject) => {
        promise.then(response => {
            const data = response.data;

            if (!data || !data.success || data.result === undefined || data.result === null) {
                reject({ message: defaultErrorMessage });
                return;
            }

            resolve(data.result);
        }).catch(error => {
            logger.error(logMessage, error);

            if (error.response && error.response.data && error.response.data.errorMessage) {
                reject({ error: error.response.data });
            } else if (!error.processed) {
                reject({ message: defaultErrorMessage });
            } else {
                reject(error);
            }
        });
    });
}
