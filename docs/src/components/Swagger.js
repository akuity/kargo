import React from 'react';
import { Provider } from 'react-redux';
import { configureStore } from '@reduxjs/toolkit';
import SwaggerUI from 'swagger-ui-react';
import 'swagger-ui-react/swagger-ui.css';

const store = configureStore({ reducer: {} });

export default function SwaggerDemo() {
    return (
        <Provider store={store}>
            <SwaggerUI url="/swagger.yaml" />    
        </Provider>
    );
}