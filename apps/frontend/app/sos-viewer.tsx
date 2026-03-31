import React, { useState, useEffect, useRef } from 'react';
import {
    View,
    Text,
    StyleSheet,
    TouchableOpacity,
    ActivityIndicator,
    Dimensions,
} from 'react-native';
import { router, useLocalSearchParams } from 'expo-router';
import { Ionicons } from '@expo/vector-icons';
import { SafeAreaView } from 'react-native-safe-area-context';
import { Colors, Spacing, BorderRadius, FontSizes } from '@/constants/theme';
import {
    connectToSOSViewerStream,
    SOSViewerStream,
    SOSLocationEvent,
    LocationPing,
} from '@/services/sos';

let MapView: any = null;
let Marker: any = null;
let PROVIDER_GOOGLE: any = null;
try {
    const maps = require('react-native-maps');
    MapView = maps.default;
    Marker = maps.Marker;
    PROVIDER_GOOGLE = maps.PROVIDER_GOOGLE;
} catch {
    MapView = null;
    Marker = null;
}

const { width, height } = Dimensions.get('window');

type ConnectionStatus = 'connecting' | 'connected' | 'error' | 'ended';

export default function SOSViewerScreen() {
    const params = useLocalSearchParams<{ token?: string; contactName?: string; lat?: string; lng?: string }>();
    const viewerToken = params.token || '';
    const contactName = params.contactName || 'Contact';
    const initialLatitude = params.lat ? Number.parseFloat(params.lat) : undefined;
    const initialLongitude = params.lng ? Number.parseFloat(params.lng) : undefined;

    const [status, setStatus] = useState<ConnectionStatus>('connecting');
    const [currentLocation, setCurrentLocation] = useState<LocationPing | null>(
        Number.isFinite(initialLatitude) && Number.isFinite(initialLongitude)
            ? {
                lat: initialLatitude as number,
                lng: initialLongitude as number,
                ts: new Date().toISOString(),
            }
            : null
    );
    const [locationHistory, setLocationHistory] = useState<LocationPing[]>(
        Number.isFinite(initialLatitude) && Number.isFinite(initialLongitude)
            ? [{
                lat: initialLatitude as number,
                lng: initialLongitude as number,
                ts: new Date().toISOString(),
            }]
            : []
    );
    const [sessionId, setSessionId] = useState<string | null>(null);
    const [lastUpdate, setLastUpdate] = useState<Date | null>(null);
    const [errorMessage, setErrorMessage] = useState<string | null>(null);

    const streamRef = useRef<SOSViewerStream | null>(null);
    const mapRef = useRef<any>(null);

    useEffect(() => {
        if (!viewerToken) {
            setStatus('error');
            setErrorMessage('No viewer token provided');
            return;
        }

        const handleEvent = (event: SOSLocationEvent) => {
            switch (event.type) {
                case 'ready':
                    setStatus('connected');
                    setSessionId(event.session_id || null);
                    break;

                case 'location':
                    if (event.location) {
                        setCurrentLocation(event.location);
                        setLocationHistory((prev) => [...prev, event.location!]);
                        setLastUpdate(new Date());

                        // Animate map to new location
                        if (mapRef.current && event.location) {
                            mapRef.current.animateToRegion({
                                latitude: event.location.lat,
                                longitude: event.location.lng,
                                latitudeDelta: 0.005,
                                longitudeDelta: 0.005,
                            }, 500);
                        }
                    }
                    break;

                case 'ended':
                    setStatus('ended');
                    break;
            }
        };

        const handleError = (error: Error) => {
            console.error('SOS Viewer stream error:', error);
            setStatus('error');
            const message = error.message.toLowerCase();
            if (
                message.includes('viewer grant revoked') ||
                message.includes('viewer grant expired') ||
                message.includes('viewer token is invalid')
            ) {
                setErrorMessage('This SOS link is no longer valid. Open the latest alert again.');
                return;
            }

            setErrorMessage(error.message || 'Connection failed');
        };

        streamRef.current = connectToSOSViewerStream(viewerToken, handleEvent, handleError);

        return () => {
            streamRef.current?.close();
        };
    }, [viewerToken]);

    const formatTime = (date: Date) => {
        return date.toLocaleTimeString('en-US', {
            hour: '2-digit',
            minute: '2-digit',
            second: '2-digit',
        });
    };

    const getTimeSinceUpdate = () => {
        if (!lastUpdate) return 'Waiting for location...';
        const seconds = Math.floor((Date.now() - lastUpdate.getTime()) / 1000);
        if (seconds < 5) return 'Just now';
        if (seconds < 60) return `${seconds}s ago`;
        return `${Math.floor(seconds / 60)}m ago`;
    };

    const handleClose = () => {
        streamRef.current?.close();
        router.back();
    };

    const renderStatusBanner = () => {
        switch (status) {
            case 'connecting':
                return (
                    <View style={[styles.statusBanner, { backgroundColor: Colors.orange }]}>
                        <ActivityIndicator size="small" color={Colors.white} />
                        <Text style={styles.statusText}>Connecting to live stream...</Text>
                    </View>
                );
            case 'connected':
                return (
                    <View style={[styles.statusBanner, { backgroundColor: Colors.sosRed }]}>
                        <View style={styles.liveDot} />
                        <Text style={styles.statusText}>LIVE - SOS Active</Text>
                    </View>
                );
            case 'ended':
                return (
                    <View style={[styles.statusBanner, { backgroundColor: Colors.success }]}>
                        <Ionicons name="checkmark-circle" size={18} color={Colors.white} />
                        <Text style={styles.statusText}>SOS Session Ended - User is Safe</Text>
                    </View>
                );
            case 'error':
                return (
                    <View style={[styles.statusBanner, { backgroundColor: Colors.red }]}>
                        <Ionicons name="warning" size={18} color={Colors.white} />
                        <Text style={styles.statusText}>{errorMessage || 'Connection Error'}</Text>
                    </View>
                );
        }
    };

    return (
        <SafeAreaView style={styles.container} edges={['top']}>
            {/* Header */}
            <View style={styles.header}>
                <TouchableOpacity onPress={handleClose} style={styles.closeButton}>
                    <Ionicons name="close" size={28} color={Colors.white} />
                </TouchableOpacity>
                <View style={styles.headerInfo}>
                    <Text style={styles.headerTitle}>SOS Alert</Text>
                    <Text style={styles.headerSubtitle}>{contactName} needs help</Text>
                </View>
                <View style={{ width: 40 }} />
            </View>

            {/* Status Banner */}
            {renderStatusBanner()}

            {/* Map */}
            <View style={styles.mapContainer}>
                {MapView && currentLocation ? (
                    <MapView
                        ref={mapRef}
                        provider={PROVIDER_GOOGLE}
                        style={StyleSheet.absoluteFillObject}
                        initialRegion={{
                            latitude: currentLocation.lat,
                            longitude: currentLocation.lng,
                            latitudeDelta: 0.005,
                            longitudeDelta: 0.005,
                        }}
                    >
                        {/* Current Location Marker */}
                        <Marker
                            coordinate={{
                                latitude: currentLocation.lat,
                                longitude: currentLocation.lng,
                            }}
                            title={contactName}
                            description="Current location"
                        >
                            <View style={styles.sosMarker}>
                                <View style={styles.sosMarkerPulse} />
                                <Ionicons name="warning" size={20} color={Colors.white} />
                            </View>
                        </Marker>
                    </MapView>
                ) : status === 'connecting' ? (
                    <View style={styles.mapPlaceholder}>
                        <ActivityIndicator size="large" color={Colors.purple} />
                        <Text style={styles.mapPlaceholderText}>
                            Connecting to live location...
                        </Text>
                    </View>
                ) : status === 'error' ? (
                    <View style={styles.mapPlaceholder}>
                        <Ionicons name="location-outline" size={64} color={Colors.grayDark} />
                        <Text style={styles.mapPlaceholderText}>
                            Unable to load location
                        </Text>
                    </View>
                ) : (
                    <View style={styles.mapPlaceholder}>
                        <Ionicons name="location-outline" size={64} color={Colors.grayDark} />
                        <Text style={styles.mapPlaceholderText}>
                            Waiting for location data...
                        </Text>
                    </View>
                )}
            </View>

            {/* Info Panel */}
            <View style={styles.infoPanel}>
                <View style={styles.infoRow}>
                    <View style={styles.infoItem}>
                        <Text style={styles.infoLabel}>Last Update</Text>
                        <Text style={styles.infoValue}>{getTimeSinceUpdate()}</Text>
                    </View>
                    <View style={styles.infoDivider} />
                    <View style={styles.infoItem}>
                        <Text style={styles.infoLabel}>Location Updates</Text>
                        <Text style={styles.infoValue}>{locationHistory.length}</Text>
                    </View>
                </View>

                {currentLocation && (
                    <View style={styles.coordsRow}>
                        <Ionicons name="navigate" size={16} color={Colors.grayLight} />
                        <Text style={styles.coordsText}>
                            {currentLocation.lat.toFixed(6)}, {currentLocation.lng.toFixed(6)}
                        </Text>
                    </View>
                )}

                {/* Action Buttons */}
                <View style={styles.actionButtons}>
                    <TouchableOpacity
                        style={styles.actionButton}
                        onPress={() => {
                            if (currentLocation) {
                                // Open in external maps app
                                const url = `https://www.google.com/maps/search/?api=1&query=${currentLocation.lat},${currentLocation.lng}`;
                                // Linking.openURL(url);
                            }
                        }}
                    >
                        <Ionicons name="navigate-outline" size={20} color={Colors.white} />
                        <Text style={styles.actionButtonText}>Open in Maps</Text>
                    </TouchableOpacity>

                    <TouchableOpacity
                        style={[styles.actionButton, { backgroundColor: Colors.sosRed }]}
                        onPress={() => {
                            // Could trigger emergency call
                        }}
                    >
                        <Ionicons name="call" size={20} color={Colors.white} />
                        <Text style={styles.actionButtonText}>Call Emergency</Text>
                    </TouchableOpacity>
                </View>
            </View>
        </SafeAreaView>
    );
}

const styles = StyleSheet.create({
    container: {
        flex: 1,
        backgroundColor: Colors.bgDark,
    },
    header: {
        flexDirection: 'row',
        alignItems: 'center',
        justifyContent: 'space-between',
        paddingHorizontal: Spacing.lg,
        paddingVertical: Spacing.md,
    },
    closeButton: {
        width: 40,
        height: 40,
        justifyContent: 'center',
        alignItems: 'center',
    },
    headerInfo: {
        alignItems: 'center',
    },
    headerTitle: {
        fontSize: FontSizes.lg,
        fontWeight: '800',
        color: Colors.sosRed,
    },
    headerSubtitle: {
        fontSize: FontSizes.sm,
        color: Colors.grayLight,
        marginTop: 2,
    },
    statusBanner: {
        flexDirection: 'row',
        alignItems: 'center',
        justifyContent: 'center',
        paddingVertical: Spacing.sm,
        gap: Spacing.sm,
    },
    liveDot: {
        width: 10,
        height: 10,
        borderRadius: 5,
        backgroundColor: Colors.white,
    },
    statusText: {
        color: Colors.white,
        fontSize: FontSizes.sm,
        fontWeight: '700',
    },
    mapContainer: {
        flex: 1,
        margin: Spacing.lg,
        borderRadius: BorderRadius.lg,
        overflow: 'hidden',
        borderWidth: 3,
        borderColor: Colors.sosRed,
    },
    mapPlaceholder: {
        flex: 1,
        justifyContent: 'center',
        alignItems: 'center',
        backgroundColor: Colors.cardBg,
        gap: Spacing.md,
    },
    mapPlaceholderText: {
        fontSize: FontSizes.sm,
        color: Colors.grayLight,
    },
    sosMarker: {
        width: 44,
        height: 44,
        borderRadius: 22,
        backgroundColor: Colors.sosRed,
        justifyContent: 'center',
        alignItems: 'center',
        borderWidth: 3,
        borderColor: Colors.white,
    },
    sosMarkerPulse: {
        position: 'absolute',
        width: 60,
        height: 60,
        borderRadius: 30,
        backgroundColor: 'rgba(239, 68, 68, 0.3)',
    },
    infoPanel: {
        backgroundColor: Colors.cardBg,
        borderTopLeftRadius: BorderRadius.xl,
        borderTopRightRadius: BorderRadius.xl,
        paddingHorizontal: Spacing.xl,
        paddingVertical: Spacing.lg,
        paddingBottom: Spacing.xxl,
        borderWidth: 1,
        borderColor: Colors.border,
    },
    infoRow: {
        flexDirection: 'row',
        alignItems: 'center',
        marginBottom: Spacing.md,
    },
    infoItem: {
        flex: 1,
        alignItems: 'center',
    },
    infoLabel: {
        fontSize: FontSizes.xs,
        color: Colors.grayLight,
        textTransform: 'uppercase',
        marginBottom: 4,
    },
    infoValue: {
        fontSize: FontSizes.lg,
        fontWeight: '800',
        color: Colors.white,
    },
    infoDivider: {
        width: 1,
        height: 40,
        backgroundColor: Colors.border,
    },
    coordsRow: {
        flexDirection: 'row',
        alignItems: 'center',
        justifyContent: 'center',
        gap: Spacing.sm,
        marginBottom: Spacing.lg,
        paddingVertical: Spacing.sm,
        backgroundColor: Colors.bgDark,
        borderRadius: BorderRadius.sm,
    },
    coordsText: {
        fontSize: FontSizes.sm,
        color: Colors.grayLight,
        fontFamily: 'monospace',
    },
    actionButtons: {
        flexDirection: 'row',
        gap: Spacing.md,
    },
    actionButton: {
        flex: 1,
        flexDirection: 'row',
        alignItems: 'center',
        justifyContent: 'center',
        gap: Spacing.sm,
        backgroundColor: Colors.purple,
        paddingVertical: Spacing.md,
        borderRadius: BorderRadius.full,
    },
    actionButtonText: {
        fontSize: FontSizes.sm,
        fontWeight: '700',
        color: Colors.white,
    },
});
