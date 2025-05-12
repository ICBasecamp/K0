"use client";

import { useState, useEffect } from "react";
import axios from "axios";

export const Terminal = ({ logs, isLoading }: { logs: string, isLoading: boolean }) => {
    return (
        <div className="flex flex-col bg-neutral-950 rounded-lg overflow-hidden">
            <div className="flex items-center px-4 py-2 bg-neutral-800 border-b border-neutral-800">
                <div className="flex space-x-2 mr-4">
                    <div className="w-3 h-3 rounded-full bg-[#ff5f56]"></div>
                    <div className="w-3 h-3 rounded-full bg-[#ffbd2e]"></div>
                    <div className="w-3 h-3 rounded-full bg-[#27c93f]"></div>
                </div>
                <div className="text-sm text-gray-400 font-medium">
                    Terminal
                </div>
            </div>
            <div className="flex flex-col bg-neutral-950 text-green-400 p-4 font-mono h-[400px] overflow-y-auto whitespace-pre-wrap">
                {isLoading ? (
                    <div className="flex items-center space-x-2">
                        <div className="animate-spin rounded-full h-4 w-4 border-t-2 border-b-2 border-green-400"></div>
                        <span>Starting container...</span>
                    </div>
                ) : (
                    logs || <span className="text-gray-500">No logs available</span>
                )}
            </div>
        </div>
    );
};

